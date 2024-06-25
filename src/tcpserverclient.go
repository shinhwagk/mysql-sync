package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"

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
	ClientFree = iota
	ClientBusy
)

type AckMessage struct {
	BatchID int
}

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

	SendBatch uint
	State     int // 0 free 1 busy

	SendError error

	IsActive bool
}

func NewTcpServerClient(logLevel int, name string, conn net.Conn) (*TCPServerClient, error) {
	var buf bytes.Buffer

	multiWriter := io.MultiWriter(conn, &buf)
	zstdWriter, err := zstd.NewWriter(multiWriter)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &TCPServerClient{
		Logger: NewLogger(logLevel, fmt.Sprintf("%s -- tcp server client(%s)", name, conn.RemoteAddr().String())),

		Name:   name,
		ctx:    ctx,
		cancel: cancel,

		conn: conn,

		encoder:           gob.NewEncoder(zstdWriter),
		encoderBuffer:     &buf,
		encoderZstdWriter: zstdWriter,
		decoder:           gob.NewDecoder(conn),

		SendBatch: 0,
		State:     ClientFree,
		SendError: nil,

		IsActive: false,
	}, nil
}

func (tsc *TCPServerClient) SetClose() {
	tsc.cancel()
}

func (tsc *TCPServerClient) Cleanup() error {
	<-tsc.ctx.Done()

	if err := tsc.encoderZstdWriter.Close(); err != nil {
		return fmt.Errorf("zstd writer close err" + err.Error())
	}
	tsc.Logger.Info("zstd write closed.")

	if err := tsc.conn.Close(); err != nil {
		return err
	}
	tsc.Logger.Info("conn closed.")

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

		tsc.Logger.Debug("Client received batch: %d successfully.", ack.BatchID)
		tsc.SendBatch = uint(ack.BatchID)
		tsc.State = ClientFree
	}
}

func (tsc *TCPServerClient) Running() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		tsc.receivedSignal()
		tsc.Logger.Info("tcp from client(%s) handler close.", tsc.conn.RemoteAddr().String())
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

func (tsc *TCPServerClient) pushClient(batchID uint, mos []MysqlOperation) error {
	signal1 := Signal1{BatchID: batchID, Mos: mos, Count: len(mos)}
	tsc.State = ClientBusy
	tsc.SendBatch = batchID
	tsc.SendError = nil
	fmt.Println("sendsendsendsendsendsendsend", batchID)

	if err := tsc.sendOperations(signal1); err != nil {
		fmt.Println("sebd", err.Error())
		tsc.SendError = err
	}

	fmt.Println("send ok", signal1.BatchID, len(mos))

	return nil
}
