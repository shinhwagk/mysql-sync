package main

import (
	"context"
	"encoding/gob"
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
		logger.Error("Connection: %s", err.Error())
		return nil, err
	}

	if _, err := conn.Write([]byte(destName + "\n")); err != nil {
		logger.Error("register: %s", err)
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
		tc.Logger.Debug("Receive mo(%d), moCh(%d) from tcp server.", len(operations), len(tc.moCh))

		if err := tc.encoder.Encode(Signal2{BatchID: sig.BatchID}); err != nil {
			tc.Logger.Error("Send signal: %s", err.Error())
			return
		}
		tc.Logger.Debug("Send signal 'BatchID: %d' success.", sig.BatchID)
	}
}

func (tc *TCPClient) Start(ctx context.Context) {
	tc.Logger.Info("Started.")
	defer tc.Logger.Info("Closed.")

	go func() {
		<-ctx.Done()
		tc.cancel()
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.receiveOperations()
		tc.Logger.Info("tcp from server '%s' handler close.", tc.conn.RemoteAddr().String())
		tc.cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.Cleanup()
	}()

	tc.Logger.Info("Server connected " + tc.conn.LocalAddr().String())
	wg.Wait()
	tc.Logger.Info("Server connection close." + tc.conn.LocalAddr().String())
}

func (tc *TCPClient) Cleanup() {
	<-tc.ctx.Done()

	tc.decoderZstdReader.Close()

	if err := tc.conn.Close(); err != nil {
		tc.Logger.Error("Close connection: %s", err.Error())
	}
}
