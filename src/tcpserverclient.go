package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

type Signal1 struct {
	BatchID uint
	Mos     []MysqlOperation
	Count   int
}

type Signal2 struct {
	BatchID uint
}

const (
	ClientInit = iota
	ClientFree
	ClientBusy
	ClientScrap
)

type TCPServerClient struct {
	Logger *Logger

	Name string

	ctx    context.Context
	cancel context.CancelFunc

	conn net.Conn

	encoder           *gob.Encoder
	encoderBuffer     *bytes.Buffer
	encoderZstdWriter *zstd.Encoder
	decoder           *gob.Decoder

	SendBatchID uint
	State       int // 0 free 1 busy 2 scrap

	SendError     error
	SendMos       []MysqlOperation
	SendTimestamp time.Time

	metricCh chan<- MetricUnit
}

func NewTcpServerClient1(logLevel int, name string, metricCh chan<- MetricUnit) *TCPServerClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPServerClient{
		Logger:      NewLogger(logLevel, fmt.Sprintf("tcp client(%s)", name)),
		Name:        name,
		ctx:         ctx,
		cancel:      cancel,
		SendBatchID: 0,
		State:       ClientScrap,
		metricCh:    metricCh,
	}
}

// func NewTcpServerClient(logLevel int, name string, conn net.Conn) (*TCPServerClient, error) {
// 	var buf bytes.Buffer

// 	multiWriter := io.MultiWriter(conn, &buf)
// 	zstdWriter, err := zstd.NewWriter(multiWriter)

// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx, cancel := context.WithCancel(context.Background())
// 	return &TCPServerClient{
// 		Logger: NewLogger(logLevel, fmt.Sprintf("%s -- tcp server client(%s)", name, conn.RemoteAddr().String())),

// 		Name:   name,
// 		ctx:    ctx,
// 		cancel: cancel,

// 		conn: conn,

// 		encoder:           gob.NewEncoder(zstdWriter),
// 		encoderBuffer:     &buf,
// 		encoderZstdWriter: zstdWriter,
// 		decoder:           gob.NewDecoder(conn),

// 		SendBatch: 0,
// 		State:     ClientFree,
// 		SendError: nil,

// 		IsAlive: false,
// 	}, nil
// }

func (tsc *TCPServerClient) InitConn(conn net.Conn) error {
	var buf bytes.Buffer

	multiWriter := io.MultiWriter(conn, &buf)
	zstdWriter, err := zstd.NewWriter(multiWriter)

	if err != nil {
		return err
	}

	tsc.ctx, tsc.cancel = context.WithCancel(context.Background())
	tsc.conn = conn
	tsc.encoder = gob.NewEncoder(zstdWriter)
	tsc.encoderBuffer = &buf
	tsc.encoderZstdWriter = zstdWriter
	tsc.decoder = gob.NewDecoder(conn)

	tsc.State = ClientFree
	return nil
}

func (tsc *TCPServerClient) SetClose() {
	tsc.cancel()
}

func (tsc *TCPServerClient) Cleanup() error {
	<-tsc.ctx.Done()

	if err := tsc.encoderZstdWriter.Close(); err != nil {
		tsc.Logger.Error("zstd writer close err: %s", err.Error())
	}
	tsc.Logger.Info("zstd write closed.")

	if err := tsc.conn.Close(); err != nil {
		tsc.Logger.Error("conn close err: %s", err.Error())
	}
	tsc.Logger.Info("conn closed.")

	tsc.State = ClientScrap

	return nil
}

func (tsc *TCPServerClient) sendOperations(s Signal1) error {
	if err := tsc.encoder.Encode(s); err != nil {
		tsc.Logger.Error("Error encoding message: %s", err.Error())
		return err
	}

	if err := tsc.encoderZstdWriter.Flush(); err != nil {
		tsc.Logger.Error("Error flushing zstd writer: %s", err)
		return err
	}

	tsc.Logger.Debug("moCh -> mo -> client cache -> mo(%d) -> client ok.", 1)
	return nil
}

func (tsc *TCPServerClient) receivedSignal() {
	for {
		var ack Signal2
		if err := tsc.decoder.Decode(&ack); err != nil {
			if err == io.EOF {
				tsc.Logger.Info("tcp client close.")
			} else {
				tsc.Logger.Error("Error decoding message:" + err.Error())
			}
			tsc.Logger.Debug("Received signal", err.Error())
			return
		}

		if tsc.SendBatchID == ack.BatchID {
			tsc.State = ClientFree
			tsc.SendError = nil
			elapsed := time.Since(tsc.SendTimestamp).Milliseconds()
			tsc.Logger.Debug("Client received batch: %d, elapsed %d successfully.", ack.BatchID, elapsed)
		} else {
			tsc.Logger.Error("not resc beatch id %d,%d", tsc.SendBatchID, ack.BatchID)
		}
	}
}

func (tsc *TCPServerClient) Running() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		tsc.receivedSignal()
		tsc.Logger.Info("tcp from client(%s) handler close.", tsc.conn.RemoteAddr().String())
		tsc.cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tsc.Cleanup(); err != nil {
			tsc.Logger.Error(fmt.Sprintf("Close connection error: " + err.Error()))
		}
	}()

	tsc.Logger.Info("Receive new client(%s) ", tsc.conn.RemoteAddr().String())
	wg.Wait()
	tsc.Logger.Info("Client(%s) closed.", tsc.conn.RemoteAddr().String())
}

func (tsc *TCPServerClient) setPush(batchID uint, mos []MysqlOperation) {
	tsc.SendBatchID = batchID
	tsc.SendMos = mos
}

func (tsc *TCPServerClient) pushClient() {
	signal1 := Signal1{BatchID: tsc.SendBatchID, Mos: tsc.SendMos, Count: len(tsc.SendMos)}
	tsc.State = ClientBusy
	tsc.SendError = nil

	if err := tsc.sendOperations(signal1); err != nil {
		tsc.Logger.Error("Push mos to client error: %s", err.Error())
		tsc.SendError = err
	} else {
		tsc.metricCh <- MetricUnit{MetricTCPServerSendOperations, uint(len(tsc.SendMos)), &tsc.Name}
		tsc.metricCh <- MetricUnit{MetricTCPServerOutgoing, uint(tsc.encoderBuffer.Len()), &tsc.Name}
		tsc.encoderBuffer.Reset()
		tsc.SendTimestamp = time.Now()
		tsc.Logger.Info("Push batch(%d) mos(%d) ok.", signal1.BatchID, len(tsc.SendMos))
	}
}
