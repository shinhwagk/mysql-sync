package main

import (
	"context"
	"net"
)

type TCPClientServer struct {
	ctx      context.Context
	conn     net.Conn
	rcCh     chan int
	moCh     chan<- MysqlOperation
	gtidSets string
}

func NewTcpClientServer(ctx context.Context, conn net.Conn, moCh chan<- MysqlOperation, gtidSets string) *TCPClientServer {
	return &TCPClientServer{
		ctx:      ctx,
		conn:     conn,
		moCh:     moCh,
		rcCh:     make(chan int),
		gtidSets: gtidSets,
	}
}
