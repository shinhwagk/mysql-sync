package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
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

func (tc *TCPClient) handleConnection(ctx context.Context, tcServer *TCPClientServer) {
	var wg sync.WaitGroup

	go func() {
		<-ctx.Done()
		tcServer.SetClose()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.handleToServer(tcServer)
		tc.Logger.Info(fmt.Sprintf("tcp to server '%s' handler close.", tcServer.conn.RemoteAddr().String()))
		tcServer.SetClose()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.handleFromServer(tcServer)
		tc.Logger.Info(fmt.Sprintf("tcp from server '%s' handler close.", tcServer.conn.RemoteAddr().String()))
		tcServer.SetClose()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tcServer.Cleanup(); err != nil {
			tc.Logger.Error(fmt.Sprintf("Close connection error: " + err.Error()))
		}
	}()

	tc.Logger.Info("Server connected " + tcServer.conn.LocalAddr().String())
	wg.Wait()
	tc.Logger.Info("Server connection close." + tcServer.conn.LocalAddr().String())
}

func (tc *TCPClient) handleToServer(tcServer *TCPClientServer) {
	if err := tcServer.encoder.Encode(fmt.Sprintf("gtidsets@%s", tcServer.gtidSets)); err != nil {
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

			signal := fmt.Sprintf("receive@%d", reciveCount)
			if err := tcServer.encoder.Encode(signal); err != nil {
				tc.Logger.Error("Request operations count: %d from tcp server, error: %s.", reciveCount, err.Error())
				return
			}
			tc.Logger.Debug("Signal '%s' has been sent to the server", signal)

			tc.metricCh <- MetricUnit{Name: MetricTCPClientRequestOperations, Value: uint(reciveCount)}
		}
	}
}

func (tc *TCPClient) handleFromServer(tcServer *TCPClientServer) {
	moCacheCh := make(chan MysqlOperation, 1000000)
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

	var lastCheck time.Time = time.Now()
	var oneSecondCount int = 0

	const minUnit int = 100

	for {
		remainingCapacity := maxRcCnt - len(moCacheCh)
		tc.Logger.Debug("MoCh remaining capacity: %d.", remainingCapacity)

		rcCnt := minUnit

		duration := time.Since(lastCheck)
		if duration > time.Second {
			if oneSecondCount >= minUnit*2 {
				rcCnt = (oneSecondCount / int(duration.Seconds()) / minUnit) * minUnit
			}
			oneSecondCount = 0
			lastCheck = time.Now()
		}

		if remainingCapacity < minUnit || float64(remainingCapacity)/float64(maxRcCnt) <= 0.2 {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		if rcCnt > 0 {
			tcServer.rcCh <- rcCnt
			tc.Logger.Debug("Requesting a batch mo: %d from tcp server.", rcCnt)

			receiveCnt := 0

			for rcCnt > 0 {
				var operations []MysqlOperation
				if err := tcServer.decoder.Decode(&operations); err != nil {
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
					oneSecondCount += 1
					receiveCnt += 1
					tc.metricCh <- MetricUnit{Name: MetricTCPClientReceiveOperations, Value: 1}
				}
				tc.Logger.Debug("Receive mo: %d from tcp server.", len(operations))
			}
			tc.Logger.Debug("Receive a batch mo: %d from tcp server.", receiveCnt)
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

	if tcServer, err := NewTcpClientServer(conn, moCh, gtidsets); err != nil {
		tc.Logger.Error("Create Client(%s) error: %s", conn.LocalAddr().String(), err.Error())
	} else {
		tc.handleConnection(ctx, tcServer)
	}
}
