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
	ClientFree int = iota
	ClientBusy
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
	State       int

	SendError     error
	SendMos       []MysqlOperation
	SendTimestamp time.Time

	metricCh chan<- MetricUnit

	Dead bool
}

func NewTcpServerClient(logLevel int, name string, metricCh chan<- MetricUnit, conn net.Conn) (*TCPServerClient, error) {
	var buf bytes.Buffer

	multiWriter := io.MultiWriter(conn, &buf)
	zstdWriter, err := zstd.NewWriter(multiWriter)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TCPServerClient{
		Logger:            NewLogger(logLevel, fmt.Sprintf("tcp-client(%s,%s)", name, conn.RemoteAddr().String())),
		Name:              name,
		ctx:               ctx,
		cancel:            cancel,
		SendBatchID:       0,
		State:             ClientFree,
		metricCh:          metricCh,
		Dead:              false,
		conn:              conn,
		encoder:           gob.NewEncoder(zstdWriter),
		encoderBuffer:     &buf,
		encoderZstdWriter: zstdWriter,
		decoder:           gob.NewDecoder(conn),
	}, nil
}

func (tsc *TCPServerClient) Cleanup() {
	tsc.Logger.Info("Cleanup started.")
	defer tsc.Logger.Info("Cleanup closed.")

	if err := tsc.encoderZstdWriter.Close(); err != nil {
		tsc.Logger.Error("Zstd writer close: %s", err.Error())
	}
	tsc.Logger.Info("Zstd writer closed.")

	if err := tsc.conn.Close(); err != nil {
		tsc.Logger.Error("Conn close: %s", err.Error())
	}
	tsc.Logger.Info("Conn closed.")
}

func (tsc *TCPServerClient) sendOperations(s Signal1) error {
	if err := tsc.encoder.Encode(s); err != nil {
		tsc.Logger.Error("Encoding message: %s", err.Error())
		return err
	}

	if err := tsc.encoderZstdWriter.Flush(); err != nil {
		tsc.Logger.Error("Zstd writer flushing: %s", err.Error())
		return err
	}

	return nil
}

func (tsc *TCPServerClient) receivedSignal() {
	tsc.Logger.Info("received signal started.")
	defer tsc.Logger.Info("received signal closed.")

	for {
		var ack Signal2
		if err := tsc.decoder.Decode(&ack); err != nil {
			if err == io.EOF {
				tsc.Logger.Info("Tcp client close.")
			} else {
				tsc.Logger.Error("Decoding message: %s", err.Error())
			}
			tsc.Logger.Error("Received signal: %s", err.Error())
			return
		}

		if tsc.SendBatchID == ack.BatchID {
			tsc.State = ClientFree
			tsc.SendError = nil
			elapsed := time.Since(tsc.SendTimestamp).Milliseconds()
			tsc.metricCh <- MetricUnit{Name: MetricTCPServerSendDelay, Value: uint(elapsed)}
			tsc.Logger.Debug("Client received batch(%d), elapsed ms(%d) successfully.", ack.BatchID, elapsed)
		} else {
			tsc.Logger.Error("Not received batch(%d), %d", tsc.SendBatchID, ack.BatchID)
		}
	}
}

func (tsc *TCPServerClient) Start() {
	tsc.Logger.Info("Started.")
	defer tsc.Logger.Info("Closed.")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		tsc.receivedSignal()
		tsc.cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-tsc.ctx.Done()
		tsc.Cleanup()
	}()

	wg.Wait()
	tsc.Dead = true
}

func (tsc *TCPServerClient) SetPush(batchID uint, mos []MysqlOperation) {
	tsc.SendBatchID = batchID
	tsc.SendMos = mos
}

func (tsc *TCPServerClient) ClientPush() {
	signal1 := Signal1{BatchID: tsc.SendBatchID, Mos: tsc.SendMos, Count: len(tsc.SendMos)}
	tsc.State = ClientBusy
	tsc.SendError = nil

	if err := tsc.sendOperations(signal1); err != nil {
		tsc.Logger.Error("Push mos to client: %s", err.Error())
		tsc.SendError = err
	} else {
		tsc.Logger.Info("Push batch(%d) mos(%d) bytes(%d) ok.", signal1.BatchID, len(tsc.SendMos), tsc.encoderBuffer.Len())
		tsc.metricCh <- MetricUnit{Name: MetricTCPServerSendOperations, Value: uint(len(tsc.SendMos))}
		tsc.metricCh <- MetricUnit{Name: MetricTCPServerOutgoing, Value: uint(tsc.encoderBuffer.Len())}
		tsc.encoderBuffer.Reset()
		tsc.SendTimestamp = time.Now()
	}
}
