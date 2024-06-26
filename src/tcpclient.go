package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"

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

	gob.Register(Signal1{})
	gob.Register(Signal2{})
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

	moCh chan<- MysqlOperation
}

func NewTCPClient(logLevel int, serverAddress string, destName string, moCh chan<- MysqlOperation, metricCh chan<- MetricUnit) (*TCPClient, error) {
	logger := NewLogger(logLevel, "tcp client")
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		logger.Error("connection error: " + err.Error())
		return nil, err
	}

	fmt.Fprintln(conn, destName)

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

		moCh: moCh,
	}, nil
}

func (tc *TCPClient) receiveOperations() {
	for {
		var sig Signal1
		if err := tc.decoder.Decode(&sig); err != nil {
			if err == io.ErrUnexpectedEOF {
				tc.Logger.Info("tcp server close, unexpected eof.")
			} else if err == io.EOF {
				tc.Logger.Info("tcp server close, eof.")
			} else {
				tc.Logger.Error("Error decoding message:" + err.Error())
			}
			return
		}

		operations := sig.Mos
		for _, oper := range operations {
			tc.moCh <- oper
			tc.metricCh <- MetricUnit{Name: MetricTCPClientReceiveOperations, Value: 1}
		}
		tc.Logger.Debug("Receive mo: %d from tcp server.", len(operations))

		if err := tc.SendSignal(sig.BatchID); err != nil {
			fmt.Println("sdsfdfsd", err.Error())
			return
		}
		fmt.Println("batch", sig.BatchID, len(sig.Mos))
	}
}

func (tc *TCPClient) handleServer() {
	var wg sync.WaitGroup

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
		if err := tc.Cleanup(); err != nil {
			tc.Logger.Error(fmt.Sprintf("Close connection error: " + err.Error()))
		}
	}()

	tc.Logger.Info("Server connected " + tc.conn.LocalAddr().String())
	wg.Wait()
	tc.Logger.Info("Server connection close." + tc.conn.LocalAddr().String())
}

func (tc *TCPClient) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		tc.SetClose()
	}()

	tc.handleServer()
}

func (tc *TCPClient) SendSignal(batchID uint) error {
	return tc.encoder.Encode(Signal2{BatchID: batchID})
}

func (tcs *TCPClient) Cleanup() error {
	<-tcs.ctx.Done()

	tcs.decoderZstdReader.Close()

	if err := tcs.conn.Close(); err != nil {
		return err
	}

	return nil
}

func (tcs *TCPClient) SetClose() {
	tcs.cancel()
}
