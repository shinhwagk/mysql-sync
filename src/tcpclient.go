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
}

type TCPClient struct {
	ctx    context.Context
	cancel context.CancelFunc

	Logger        *Logger
	ServerAddress string
	metricCh      chan<- MetricUnit

	conn              net.Conn
	encoder           *gob.Encoder
	decoder           *gob.Decoder
	decoderZstdReader *zstd.Decoder

	rcCh     chan int //receive count
	moCh     chan<- MysqlOperation
	gtidSets string

	moCacheCh chan MysqlOperation
}

func NewTCPClient(logLevel int, serverAddress string, moCh chan<- MysqlOperation, gtidSets string, metricCh chan<- MetricUnit) (*TCPClient, error) {
	logger := NewLogger(logLevel, "tcp client")
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		logger.Error("connection error: " + err.Error())
		return nil, err
	}

	encoder := gob.NewEncoder(conn)

	zstdReader, err := zstd.NewReader(conn)
	if err != nil {
		return nil, err
	}
	decoder := gob.NewDecoder(zstdReader)

	ctx, cancel := context.WithCancel(context.Background())

	return &TCPClient{
		ctx:    ctx,
		cancel: cancel,

		Logger:            logger,
		ServerAddress:     serverAddress,
		conn:              conn,
		encoder:           encoder,
		decoder:           decoder,
		decoderZstdReader: zstdReader,
		metricCh:          metricCh,

		moCh:     moCh,
		rcCh:     make(chan int),
		gtidSets: gtidSets,

		moCacheCh: make(chan MysqlOperation, 100000),
	}, nil
}

func (tc *TCPClient) flowControl() {
	fn := func(lastSecondCount, maxRcCnt, minRcCnt int) {
		remainingCapacity := maxRcCnt - len(tc.moCacheCh)
		tc.Logger.Debug("MoCh remaining capacity: %d/%d.", remainingCapacity, maxRcCnt)

		reciveCount := max(minRcCnt, (lastSecondCount/minRcCnt)*minRcCnt)

		if remainingCapacity < minRcCnt || float64(remainingCapacity)/float64(maxRcCnt) <= 0.2 {
			time.Sleep(time.Millisecond * 100)
			return
		}

		if reciveCount > 0 {
			signal := fmt.Sprintf("receive@%d", reciveCount)
			if err := tc.SendSignal(signal); err != nil {
				tc.Logger.Error("Request operations count: %d from tcp server, error: %s.", reciveCount, err.Error())
				return
			} else {
				tc.Logger.Debug("Signal '%s' has been sent to the server", signal)
			}
			tc.rcCh <- reciveCount
			tc.metricCh <- MetricUnit{Name: MetricTCPClientRequestOperations, Value: uint(reciveCount)}
		}
	}

	maxRcCnt := cap(tc.moCacheCh)
	const minRcCnt int = 100

	lastSecondCount := 0

	ticker1 := time.NewTicker(time.Second * 1)
	defer ticker1.Stop()

	ticker2 := time.NewTicker(time.Millisecond * 100)
	defer ticker2.Stop()

	for {
		select {
		case <-tc.ctx.Done():
			tc.Logger.Info("Context cancelled, stopping handleFromServer loop.")
			return
		case <-ticker1.C:
			fmt.Println("asdfsdfsdfsfsdfsdfsdfsdfsdf", lastSecondCount)
			fn(lastSecondCount, maxRcCnt, minRcCnt)
			lastSecondCount = 0
		case <-ticker2.C:
			fn(0, maxRcCnt, minRcCnt)
		case mo, ok := <-tc.moCacheCh:
			if !ok {
				tc.Logger.Info("channel 'moCacheCh' closed.")
				return
			}
			tc.moCh <- mo
			lastSecondCount++
		}
	}
}

func (tc *TCPClient) receiveOperations() {
	for {
		select {
		case <-tc.ctx.Done():
			tc.Logger.Info("Context cancelled, stopping handleFromServer loop.")
			return
		case rcCnt, ok := <-tc.rcCh:
			if !ok {
				tc.Logger.Info("channel 'reciveCountCh' closed.")
				return
			}
			tc.Logger.Debug("Requesting a batch mo: %d from tcp server.", rcCnt)

			receiveCnt := 0

			for rcCnt != receiveCnt {
				var operations []MysqlOperation
				if err := tc.decoder.Decode(&operations); err != nil {
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
					tc.moCacheCh <- oper
					receiveCnt++
					tc.metricCh <- MetricUnit{Name: MetricTCPClientReceiveOperations, Value: 1}
				}
				tc.Logger.Debug("Receive mo: %d from tcp server.", len(operations))
			}
			tc.Logger.Debug("Receive a batch mo: %d from tcp server.", receiveCnt)
		case <-time.After(time.Second * 1):
		}
	}
}

func (tc *TCPClient) Start(ctx context.Context) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		tc.SetClose()
	}()

	go func() {
		defer wg.Done()
		if err := tc.SendSignal(fmt.Sprintf("gtidsets@%s", tc.gtidSets)); err != nil {
			tc.Logger.Info("Requesting operations from server with gtidsets '%s' error: %s.", tc.gtidSets, err.Error())
			tc.SetClose()
		}
		tc.Logger.Info("Requesting operations from server with gtidsets '%s'.", tc.gtidSets)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.receiveOperations()
		tc.Logger.Info(fmt.Sprintf("tcp from server '%s' handler close.", tc.conn.RemoteAddr().String()))
		tc.SetClose()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.flowControl()
		tc.Logger.Info(fmt.Sprintf("tcp to server '%s' handler close.", tc.conn.RemoteAddr().String()))
		tc.SetClose()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tc.Cleanup(); err != nil {
			tc.Logger.Error(fmt.Sprintf("Close connection error: " + err.Error()))
		}
	}()

	tc.Logger.Info("Server connected " + tc.conn.LocalAddr().String())
	wg.Wait()
	tc.Logger.Info("Server connection close." + tc.conn.LocalAddr().String())
}

func (tc *TCPClient) SendSignal(signal string) error {
	return tc.encoder.Encode(signal)
}

func (tcs *TCPClient) Cleanup() error {
	<-tcs.ctx.Done()

	select {
	case <-tcs.rcCh:
		// ts.Logger.Debug(fmt.Sprintf("moCh -> mo -> client cache(%s) ok.", client.conn.RemoteAddr().String()))
	case <-time.After(time.Second * 1):
		fmt.Println("channel rcCh clean.")
	}

	select {
	case <-tcs.moCacheCh:
		// ts.Logger.Debug(fmt.Sprintf("moCh -> mo -> client cache(%s) ok.", client.conn.RemoteAddr().String()))
	case <-time.After(time.Second * 1):
		fmt.Println("channel moCacheCh clean.")
	}

	tcs.decoderZstdReader.Close()

	if err := tcs.conn.Close(); err != nil {
		return err
	}

	close(tcs.rcCh)
	close(tcs.moCacheCh)

	return nil
}

func (tcs *TCPClient) SetClose() {
	tcs.cancel()
}
