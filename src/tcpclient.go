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

func (tc *TCPClient) sendServer(encoder *gob.Encoder, controlSignal string) error {
	if err := encoder.Encode(&controlSignal); err != nil {
		return err
	}
	return nil
}

func (tc *TCPClient) SendSignalReceive(encoder *gob.Encoder, reciveCount int) error {
	signal := fmt.Sprintf("receive@%d", reciveCount)
	fmt.Println(signal)
	if err := tc.sendServer(encoder, signal); err != nil {
		tc.Logger.Error(fmt.Sprintf("send control signal '%s' to server error: %s", signal, err.Error()))
		return err
	} else {
		tc.Logger.Debug(fmt.Sprintf("send control signal '%s' to server."))
	}
	return nil
}

func (tc *TCPClient) handleConnection(ctx context.Context, conn net.Conn, moCh chan<- MysqlOperation, gtidset string) {
	rcCh := make(chan int)
	defer close(rcCh)

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	go func() {
		rcCh <- cap(moCh)
		fmt.Println("init moch ", cap(moCh))
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.handleToServer(ctx, conn, rcCh, gtidset)
		tc.Logger.Info(fmt.Sprintf("tcp to server(%s) handler close.", conn.RemoteAddr().String()))
		// cancel()

	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.handleFromServer(ctx, conn, rcCh, moCh)
		tc.Logger.Info(fmt.Sprintf("tcp from server(%s) handler close.", conn.RemoteAddr().String()))
		// cancel()

	}()
	wg.Wait()
}

func (tc *TCPClient) handleToServer(ctx context.Context, conn net.Conn, rcCh <-chan int, gtidset string) {
	encoder := gob.NewEncoder(conn)

	if err := tc.sendServer(encoder, fmt.Sprintf("gtidset@%s", gtidset)); err != nil {
		tc.Logger.Error("sender gtidset faile " + err.Error())
		return
	}

	fmt.Println("xxxx", fmt.Sprintf("gtidset@%s", gtidset))

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 1):
		case rc, ok := <-rcCh:
			if !ok {
				tc.Logger.Info("ReceiveCountCh")
				return
			}
			if err := tc.SendSignalReceive(encoder, rc); err != nil {
				return
			}
		}
	}

}

func (tc *TCPClient) handleFromServer(ctx context.Context, conn net.Conn, rcCh chan<- int, moCh chan<- MysqlOperation) {
	// conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	zstdReader, err := zstd.NewReader(conn)
	if err != nil {
		tc.Logger.Error("Error creating zstd reader:" + err.Error())
		return
	}
	defer zstdReader.Close()
	decoder := gob.NewDecoder(zstdReader)

	var count = 0

	var rcCnt = cap(moCh)

	for {
		select {
		case <-ctx.Done(): // 检查 context 是否被取消或超时
			tc.Logger.Info("Context cancelled, stopping handleFromServer loop.")
			return
		default:
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
				moCh <- oper
				count += 1
				rcCnt -= 1

				tc.metricCh <- MetricUnit{Name: MetricTCPClientOperations, Value: 1}
			}

			if rcCnt == 0 {
				rcCnt += 100
				rcCh <- 100
				// time.Sleep(time.Second * 1)
			}
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
