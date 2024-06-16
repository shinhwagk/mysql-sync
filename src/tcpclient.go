package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

func init() {
	gob.Register(MysqlOperationHeartbeat{})
	gob.Register(MysqlOperationDDLDatabase{})
	gob.Register(MysqlOperationGTID{})
	gob.Register(MysqlOperationDDLTable{})
	gob.Register(MysqlOperationDMLInsert{})
	gob.Register(MysqlOperationDMLUpdate{})
	gob.Register(MysqlOperationDMLDelete{})
	gob.Register(MysqlOperationBegin{})
	gob.Register(MysqlOperationXid{})

	// gob.Register(&DTOPause{})
	// gob.Register(&DTOResume{})
	// gob.Register(&DTOGtidSet{})

}

type TCPClient struct {
	Logger        *Logger
	ServerAddress string
	metricCh      chan<- MetricUnit
}

type TCPClientConn struct {
	ctx     context.Context
	conn    net.Conn
	rcCh    chan int
	moCh    chan<- MysqlOperation
	gtidSet string
}

func NewTCPClient(logLevel int, serverAddress string, metricCh chan<- MetricUnit) *TCPClient {
	return &TCPClient{
		Logger:        NewLogger(logLevel, "tcp client"),
		ServerAddress: serverAddress,

		metricCh: metricCh,
	}
}

func (tc *TCPClient) SendSignalReceive(encoder *gob.Encoder, reciveCount int) error {
	tc.Logger.Debug(fmt.Sprintf("Request operations count: %d from tcp server.", reciveCount))

	signal := fmt.Sprintf("receive@%d", reciveCount)
	if err := encoder.Encode(signal); err != nil {
		tc.Logger.Error(fmt.Sprintf("Request operations count: %d from tcp server, error: %s.", reciveCount, err.Error()))
		return err
	}
	return nil
}

func (tc *TCPClient) handleConnection(ctx context.Context, conn net.Conn, moCh chan<- MysqlOperation, gtidset string) {
	rcCh := make(chan int)
	defer close(rcCh)

	_ctx, _cancel := context.WithCancel(context.Background())
	defer _cancel()

	var wg sync.WaitGroup

	tcc := &TCPClientConn{_ctx, conn, rcCh, moCh, gtidset}

	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.handleToServer(tcc)
		tc.Logger.Info(fmt.Sprintf("tcp to server '%s' handler close.", conn.RemoteAddr().String()))
		_cancel()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.handleFromServer(_ctx, conn, rcCh, moCh)
		tc.Logger.Info(fmt.Sprintf("tcp from server '%s' handler close.", conn.RemoteAddr().String()))
		_cancel()
	}()
	wg.Wait()
}

func (tc *TCPClient) handleToServer(tcc *TCPClientConn) {
	encoder := gob.NewEncoder(tcc.conn)

	if err := encoder.Encode(fmt.Sprintf("gtidset@%s", tcc.gtidSet)); err != nil {
		tc.Logger.Info(fmt.Sprintf("Requesting operations from server with gtidset '%s' error: %s.", tcc.gtidSet, err.Error()))
		return
	}
	tc.Logger.Info(fmt.Sprintf("Requesting operations from server with gtidset '%s'.", tcc.gtidSet))

	for {
		select {
		case <-time.After(time.Second * 1):
		case <-tcc.ctx.Done():
			return
		case rc, ok := <-tcc.rcCh:
			if !ok {
				return
			}
			if err := tc.SendSignalReceive(encoder, rc); err != nil {
				return
			}
			tc.metricCh <- MetricUnit{Name: MetricTCPClientRequestOperations, Value: uint(rc)}
		}
	}
}

func (tc *TCPClient) handleFromServer(ctx context.Context, conn net.Conn, rcCh chan<- int, moCh chan<- MysqlOperation) {
	zstdReader, err := zstd.NewReader(conn)
	if err != nil {
		tc.Logger.Error("Error creating zstd reader:" + err.Error())
		return
	}
	defer zstdReader.Close()
	decoder := gob.NewDecoder(zstdReader)

	moCacheCh := make(chan MysqlOperation, 1000)
	defer close(moCacheCh)

	go func() {
		for {
			select {
			case <-ctx.Done():
				tc.Logger.Info("Context cancelled, stopping handleFromServer loop.")
				return
			case mo, ok := <-moCacheCh:
				if !ok {
					tc.Logger.Info("channel 'moCacheCh' closed.")
					return
				}
				moCh <- mo
			}
		}
	}()

	var maxRcCnt = cap(moCacheCh)

	for {

		remainingCapacity := maxRcCnt - len(moCacheCh)
		rcCnt := 0

		if remainingCapacity >= 100 {
			rcCnt = 100
		} else {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		if rcCnt > 0 {
			rcCh <- rcCnt

			tc.Logger.Info(fmt.Sprintf("Requesting operations: %d from tcp server.", rcCnt))

			for rcCnt > 0 {
				var operations []MysqlOperation
				if err := decoder.Decode(&operations); err != nil {
					if err == io.ErrUnexpectedEOF {
						tc.Logger.Info("tcp server close, unexpected eof.")
					} else if err == io.EOF {
						tc.Logger.Info("tcp server close, eof.")
					} else {
						tc.Logger.Error("Error decoding message:" + err.Error())
					}
					return
				}

				for _, oper := range operations {
					moCacheCh <- oper
					rcCnt -= 1
					tc.metricCh <- MetricUnit{Name: MetricTCPClientReceiveOperations, Value: 1}
				}
			}

			tc.Logger.Info(fmt.Sprintf("Receive operations: %d from tcp server.", rcCnt))
		}
	}
}

func (tc *TCPClient) Start(ctx context.Context, moCh chan<- MysqlOperation, gtidset string) {
	conn, err := net.Dial("tcp", tc.ServerAddress)
	if err != nil {
		tc.Logger.Error("connection error: " + err.Error())
		return
	}
	defer conn.Close()

	tc.handleConnection(ctx, conn, moCh, gtidset)
}
