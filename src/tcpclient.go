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

func NewTCPClient(logLevel int, serverAddress string, metricCh chan<- MetricUnit) *TCPClient {
	return &TCPClient{
		Logger:        NewLogger(logLevel, "tcp client"),
		ServerAddress: serverAddress,

		metricCh: metricCh,
	}
}

func (tc *TCPClient) SendSignalReceive(encoder *gob.Encoder, reciveCount int) error {
	signal := fmt.Sprintf("receive@%d", reciveCount)
	if err := encoder.Encode(signal); err != nil {
		tc.Logger.Error(fmt.Sprintf("Request operations count: %d from tcp server, error: %s.", reciveCount, err.Error()))
		return err
	}
	return nil
}

func (tc *TCPClient) handleConnection(tcServer *TCPClientServer) {

	var wg sync.WaitGroup

	// tcc := &TCPClientServer{_ctx, conn, rcCh, moCh, gtidsets}

	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.handleToServer(tcServer)
		tc.Logger.Info(fmt.Sprintf("tcp to server '%s' handler close.", tcServer.conn.RemoteAddr().String()))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.handleFromServer(tcServer)
		tc.Logger.Info(fmt.Sprintf("tcp from server '%s' handler close.", tcServer.conn.RemoteAddr().String()))
	}()

	wg.Add(1)
	go func() {
		select {
		case <-tcServer.ctx.Done():
			if err := tcServer.conn.Close(); err != nil {
				tc.Logger.Error(fmt.Sprintf("Close connection error: " + err.Error()))
			}
		}
	}()

	fmt.Println("Start conn " + tcServer.conn.LocalAddr().String())
	wg.Wait()
}

func (tc *TCPClient) handleToServer(tcServer *TCPClientServer) {
	encoder := gob.NewEncoder(tcServer.conn)

	if err := encoder.Encode(fmt.Sprintf("gtidsets@%s", tcServer.gtidSets)); err != nil {
		tc.Logger.Info("Requesting operations from server with gtidsets '%s' error: %s.", tcServer.gtidSets, err.Error())
		return
	}
	tc.Logger.Info("Requesting operations from server with gtidsets '%s'.", tcServer.gtidSets)

	for {
		select {
		case <-time.After(time.Second * 1):
		case <-tcServer.ctx.Done():
			return
		case reciveCount, ok := <-tcServer.rcCh:
			if !ok {
				tc.Logger.Info("channel 'reciveCountCh' closed.")
				return
			}
			tc.Logger.Info(fmt.Sprintf("Requesting operations: %d from tcp server.", 100))

			signal := fmt.Sprintf("receive@%d", reciveCount)
			if err := encoder.Encode(signal); err != nil {
				tc.Logger.Error("Request operations count: %d from tcp server, error: %s.", reciveCount, err.Error())
				return
			}
			tc.metricCh <- MetricUnit{Name: MetricTCPClientRequestOperations, Value: uint(reciveCount)}
		}
	}
}

func (tc *TCPClient) handleFromServer(tcServer *TCPClientServer) {
	zstdReader, err := zstd.NewReader(tcServer.conn)
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
			case <-time.After(time.Second * 1):
			case <-tcServer.ctx.Done():
				tc.Logger.Info("Context cancelled, stopping handleFromServer loop.")
				return
			case mo, ok := <-moCacheCh:
				if !ok {
					tc.Logger.Info("channel 'moCacheCh' closed.")
					return
				}
				tcServer.moCh <- mo
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
			tcServer.rcCh <- rcCnt

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
			tc.Logger.Info("Receive operations: %d from tcp server.", 100)
		}
	}
}

func (tc *TCPClient) Start(ctx context.Context, moCh chan<- MysqlOperation, gtidsets string) {
	conn, err := net.Dial("tcp", tc.ServerAddress)
	if err != nil {
		tc.Logger.Error("connection error: " + err.Error())
		return
	}
	defer conn.Close()

	server := NewTcpClientServer(ctx, conn, moCh, gtidsets)

	tc.handleConnection(server)
}
