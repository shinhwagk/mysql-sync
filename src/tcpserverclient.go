package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/klauspost/compress/zstd"
)

type TcpServerClient struct {
	Logger *Logger

	ctx    context.Context
	cancel context.CancelFunc

	conn net.Conn

	encoder           *gob.Encoder
	encoderBuffer     *bytes.Buffer
	encoderZstdWriter *zstd.Encoder
	decoder           *gob.Decoder

	close bool

	channel chan MysqlOperation
	rcCh    chan int
}

func NewTcpServerClient(logLevel int, conn net.Conn) (*TcpServerClient, error) {
	var buf bytes.Buffer

	multiWriter := io.MultiWriter(conn, &buf)
	zstdWriter, err := zstd.NewWriter(multiWriter)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &TcpServerClient{
		Logger: NewLogger(logLevel, fmt.Sprintf("tcp server client(%s)", conn.RemoteAddr().String())),

		ctx:    ctx,
		cancel: cancel,

		conn:    conn,
		channel: make(chan MysqlOperation),
		close:   false,
		rcCh:    make(chan int),

		encoder:           gob.NewEncoder(zstdWriter),
		encoderBuffer:     &buf,
		encoderZstdWriter: zstdWriter,
		decoder:           gob.NewDecoder(conn),
	}, nil
}

func (tsc *TcpServerClient) SetClose() {
	tsc.close = true
	tsc.cancel()
}

func (tsc *TcpServerClient) Cleanup() error {
	<-tsc.ctx.Done()

	select {
	case <-tsc.channel:
		// ts.Logger.Debug(fmt.Sprintf("moCh -> mo -> client cache(%s) ok.", client.conn.RemoteAddr().String()))
	case <-time.After(time.Second * 1):
		tsc.Logger.Info("channel moCh clean.")
	}

	select {
	case <-tsc.rcCh:
		// ts.Logger.Debug(fmt.Sprintf("moCh -> mo -> client cache(%s) ok.", client.conn.RemoteAddr().String()))
	case <-time.After(time.Second * 1):
		tsc.Logger.Info("channel rcCh clean.")
	}

	if err := tsc.encoderZstdWriter.Close(); err != nil {
		return fmt.Errorf("zstd writer close err" + err.Error())
	}
	tsc.Logger.Info("zstd write closed.")

	if err := tsc.conn.Close(); err != nil {
		return err
	}
	tsc.Logger.Info("conn closed.")

	close(tsc.rcCh)
	tsc.Logger.Info("channel rcCh closed.")

	close(tsc.channel)
	tsc.Logger.Info("channel moCh closed.")

	return nil
}

func (tsc *TcpServerClient) sendOperations(mos []MysqlOperation) error {
	if err := tsc.encoder.Encode(mos); err != nil {
		tsc.Logger.Error("Error encoding message: %s", err.Error())
		return err
	}

	if err := tsc.encoderZstdWriter.Flush(); err != nil {
		tsc.Logger.Error("Error flushing zstd writer: %s", err)
		return err
	}

	tsc.Logger.Debug("moCh -> mo -> client cache -> mo(%d) -> client ok.", len(mos))
	return nil
}
