package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"github.com/klauspost/compress/zstd"
)

type TCPClientServer struct {
	ctx    context.Context
	cancel context.CancelFunc

	conn              net.Conn
	encoder           *gob.Encoder
	decoder           *gob.Decoder
	decoderZstdReader *zstd.Decoder

	rcCh     chan int
	moCh     chan<- MysqlOperation
	gtidSets string

	moCacheCh chan MysqlOperation
}

func NewTcpClientServer(conn net.Conn, moCh chan<- MysqlOperation, gtidSets string) (*TCPClientServer, error) {
	encoder := gob.NewEncoder(conn)

	zstdReader, err := zstd.NewReader(conn)
	if err != nil {
		return nil, err
	}
	decoder := gob.NewDecoder(zstdReader)

	ctx, cancel := context.WithCancel(context.Background())
	return &TCPClientServer{
		ctx:    ctx,
		cancel: cancel,

		conn:              conn,
		encoder:           encoder,
		decoderZstdReader: zstdReader,
		decoder:           decoder,

		moCh:     moCh,
		rcCh:     make(chan int),
		gtidSets: gtidSets,

		moCacheCh: make(chan MysqlOperation, 100000),
	}, nil
}

func (tcs *TCPClientServer) Cleanup() error {
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

func (tcs *TCPClientServer) SetClose() {
	tcs.cancel()
}

func (tcs *TCPClientServer) SendSignal(signal string) error {
	return tcs.encoder.Encode(signal)
}
