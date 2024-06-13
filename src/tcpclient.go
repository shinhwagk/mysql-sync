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
	Name string

	ServerAddress string
	Logger        *Logger

	// metric   *MetricTCPClient
	metricCh chan<- MetricUnit
	// decoder *gob.Decoder
	// encoder *gob.Encoder
}

func NewTCPClient(logLevel int, serverAddress string, metricCh chan<- MetricUnit) *TCPClient {
	return &TCPClient{
		Logger:        NewLogger(logLevel, "tcp client"),
		ServerAddress: serverAddress,

		metricCh: metricCh,
	}
}

func (tc *TCPClient) Close() {

}

func (tc *TCPClient) sendServer(encoder *gob.Encoder, controlSignal string) error {
	if err := encoder.Encode(&controlSignal); err != nil {
		return err
	}
	return nil
}

func (tc *TCPClient) Stop() {

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

func (tc *TCPClient) handleConnection(conn net.Conn, moCh chan<- MysqlOperation, gtidset string) {
	rcCh := make(chan int)
	defer close(rcCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		cancel()

	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.handleFromServer(ctx, conn, rcCh, moCh)
		tc.Logger.Info(fmt.Sprintf("tcp from server(%s) handler close.", conn.RemoteAddr().String()))
		cancel()
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

	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			timer.Reset(time.Second * 1)
		case rc, ok := <-rcCh:
			if !ok {
				tc.Logger.Info("ReceiveCountCh")
				return
			}
			if err := tc.SendSignalReceive(encoder, rc); err != nil {
				return
			}
			timer.Reset(time.Second * 1)
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

	var count = 0

	var rcCnt = cap(moCh)

	for {
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
			time.Sleep(time.Second * 1)
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

	tc.handleConnection(conn, moCh, gtidset)
	// zstdReader, err := zstd.NewReader(conn)
	// if err != nil {
	// 	tc.Logger.Error("Error creating zstd reader:" + err.Error())
	// 	return
	// }
	// defer zstdReader.Close()

	// decoder := gob.NewDecoder(zstdReader)
	// encoder := gob.NewEncoder(conn)

	// if err := tc.sendServer(encoder, fmt.Sprintf("gtidset@%s", gtidset)); err != nil {
	// 	tc.Logger.Error("sender gtidset faile " + err.Error())
	// 	return
	// }

	// if err := tc.SendSignalReceive(encoder, cap(moCh)); err != nil {
	// 	return
	// }

	// var moChLen = 0
	// // moChCap := cap(moCh)
	// receive := cap(moCh)
	// var count = 0

	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		tc.Logger.Info("ctx done signal received.")
	// 		return
	// 	default:
	// 		var operations []MysqlOperation
	// 		if err := decoder.Decode(&operations); err != nil {
	// 			if err == io.ErrUnexpectedEOF {
	// 				tc.Logger.Info("tcp server close, unexpected eof.")
	// 			} else if err == io.EOF {
	// 				tc.Logger.Info("tcp server close, eof.")
	// 			} else {
	// 				tc.Logger.Error("Error decoding message:" + err.Error())
	// 			}
	// 			return
	// 		}

	// 		tc.Logger.Debug(fmt.Sprintf("receive operations count:'%d' from server.", len(operations)))

	// 		moChLen += len(operations)

	// 		count += len(operations)
	// 		fmt.Println("colunt", count)

	// 		for _, oper := range operations {
	// 			moCh <- oper
	// 			moChLen -= 1
	// 			receive -= 1
	// 			tc.metricCh <- MetricUnit{Name: MetricTCPClientOperations, Value: 1}
	// 		}

	// 		fmt.Println(fmt.Sprintf("receive operations receive:'%d' from server.", receive))
	// 		// time.Sleep(time.Second * 1000)

	// 		// if receive == 0 {
	// 		// 	if moChLen <= moChCap*20/100 {
	// 		// 		if err := tc.SendSignalReceive(encoder, uint(moChCap-moChLen)); err != nil {
	// 		// 			return
	// 		// 		}
	// 		// 		receive = moChCap - moChLen
	// 		// 		tc.Logger.Info(fmt.Sprintf("send signal receive '%d'.", uint(moChCap-moChLen)))
	// 		// 	}
	// 		// }
	// 	}
	// }
}
