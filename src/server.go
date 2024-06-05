package main

import (
	"context"
	"encoding/binary"
	"log"
	"net"
)

type TCPServer struct {
	Address string // IP地址和端口
}

func NewTCPServer(address string) *TCPServer {
	return &TCPServer{Address: address}
}

func (s *TCPServer) Start(ctx context.Context) {
	listener, err := net.Listen("tcp", s.Address)
	if err != nil {
		log.Fatal("Error listening:", err)
	}
	defer listener.Close()
	log.Println("Server listening on", s.Address)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting:", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	// 处理连接逻辑
	var startSerial int
	err := binary.Read(conn, binary.BigEndian, &startSerial)
	if err != nil {
		log.Printf("Error reading serial number: %v", err)
		return
	}
	log.Printf("Received serial number request: %d", startSerial)
	// 进一步的处理...
}

// func main() {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	server := NewTCPServer("localhost:9999")
// 	server.Start(ctx)
// }
