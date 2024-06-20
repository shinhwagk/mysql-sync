package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"

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

	channel chan MysqlOperation
	close   bool
	rcCh    chan int
}

func NewTcpServerClient(loglevel int, conn net.Conn) (*TcpServerClient, error) {
	var buf bytes.Buffer

	multiWriter := io.MultiWriter(conn, &buf)
	zstdWriter, err := zstd.NewWriter(multiWriter)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &TcpServerClient{
		Logger: NewLogger(loglevel, fmt.Sprintf("tcp server client(%s)", conn.RemoteAddr().String())),

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

	close(tsc.channel)

	for {
		_, ok := <-tsc.channel
		fmt.Println("sfsdfsdfsdfsdfsfsdf", ok)
		if !ok {
			break
		}
	}

	if err := tsc.encoderZstdWriter.Close(); err != nil {
		return fmt.Errorf("zstd writer close err" + err.Error())
	}

	if err := tsc.conn.Close(); err != nil {
		return err
	}

	return nil
}
