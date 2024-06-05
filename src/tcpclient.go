package main

import (
	"encoding/binary"
	"log"
	"net"
)

type Request struct {
	SerialNumber int
	RequestType  string
	Data         string
}

type TCPClient struct {
	ServerAddress string
	Logger        Logger
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

	err = binary.Write(conn, binary.BigEndian, serialNumber)
	if err != nil {
		log.Fatalf("Error writing serial number: %v", err)
	}
	log.Printf("Sent serial number: %d to server", serialNumber)

}

// func main() {
// 	client := NewTCPClient("localhost:9999")
// 	client.ConnectAndSend(20)
// }
