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
	ctx               context.Context
	cancel            context.CancelFunc
	Logger            *Logger
	ServerAddress     string
	metricCh          chan<- MetricUnit
	conn              net.Conn
	encoder           *gob.Encoder
	decoder           *gob.Decoder
	decoderZstdReader *zstd.Decoder
	BatchID           uint
	moCh              chan<- MysqlOperation
	// StartGtidSets     string
}

func NewTCPClient(logLevel int, serverAddress string, destName string, moCh chan<- MysqlOperation, metricCh chan<- MetricUnit, startGtidSets string) (*TCPClient, error) {
	logger := NewLogger(logLevel, "tcp-client")
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		logger.Error("Connection: %s", err.Error())
		return nil, err
	}

	initClientInfo := fmt.Sprintf("%s@%s", destName, startGtidSets)

	if _, err := conn.Write([]byte(initClientInfo + "\n")); err != nil {
		logger.Error("register: %s", err)
	} else {
		logger.Info("Send init client info: %s", initClientInfo)
	}

	encoder := gob.NewEncoder(conn)

	zstdReader, err := zstd.NewReader(conn)
	if err != nil {
		return nil, err
	}
	decoder := gob.NewDecoder(zstdReader)

	ctx, cancel := context.WithCancel(context.Background())

	return &TCPClient{
		ctx:               ctx,
		cancel:            cancel,
		Logger:            logger,
		ServerAddress:     serverAddress,
		conn:              conn,
		encoder:           encoder,
		decoder:           decoder,
		decoderZstdReader: zstdReader,
		metricCh:          metricCh,
		BatchID:           0,
		moCh:              moCh,
		// StartGtidSets:     startGtidSets,
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

		if sig.BatchID == tc.BatchID+1 && len(sig.Mos) == sig.Count {
			for _, oper := range sig.Mos {
				tc.moCh <- oper
			}
			tc.metricCh <- MetricUnit{Name: MetricTCPClientReceiveOperations, Value: uint(sig.Count)}
			tc.Logger.Debug("Receive Batch: %d, mo count: %d from tcp server.", sig.BatchID, len(sig.Mos))

			if err := tc.encoder.Encode(Signal2{BatchID: sig.BatchID}); err != nil {
				tc.Logger.Error("Send signal: %s", err.Error())
				return
			}
			tc.Logger.Debug("Send signal 'Batch: %d' success.", sig.BatchID)
			tc.BatchID = sig.BatchID
		} else {
			tc.Logger.Error("Receive Server Batch: %s, Client Batch: %s, Count: %d, Mos: %d", sig.BatchID, tc.BatchID, sig.Count, len(sig.Mos))
			return
		}
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

	wg.Wait()
}

func (tc *TCPClient) Cleanup() {
	<-tc.ctx.Done()
	tc.decoderZstdReader.Close()

	if err := tc.conn.Close(); err != nil {
		tc.Logger.Error("Close connection: %s", err.Error())
	}
}
