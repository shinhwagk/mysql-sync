package main

import (
	"encoding/binary"
	"log"
	"net"
)

type TCPClient struct {
	ServerAddress string
}

func NewTCPClient(serverAddress string) *TCPClient {
	return &TCPClient{ServerAddress: serverAddress}
}

func (c *TCPClient) ConnectAndSend(serialNumber int) {
	conn, err := net.Dial("tcp", c.ServerAddress)
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	defer conn.Close()

	// 发送序列号到服务器
	err = binary.Write(conn, binary.BigEndian, serialNumber)
	if err != nil {
		log.Fatalf("Error writing serial number: %v", err)
	}
	log.Printf("Sent serial number: %d to server", serialNumber)
	// 从此处扩展以接收和处理来自服务器的数据
}

// func main() {
// 	client := NewTCPClient("localhost:9999")
// 	client.ConnectAndSend(20)
// }
