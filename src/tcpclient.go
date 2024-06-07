package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"

	"github.com/klauspost/compress/zstd"
)

func init() {
	gob.Register(MysqlOperationHeartbeat{})
	gob.Register(MysqlOperationGTID{})
	gob.Register(MysqlOperationDDLDatabase{})

	// gob.Register(&DTOPause{})
	// gob.Register(&DTOResume{})
	// gob.Register(&DTOGtidSet{})

}

type TCPClient struct {
	Name string

	ServerAddress string
	logger        *Logger

	// decoder *gob.Decoder
	// encoder *gob.Encoder
}

func NewTCPClient(logLevel int, serverAddress string) *TCPClient {
	return &TCPClient{
		ServerAddress: serverAddress,
		logger:        NewLogger(logLevel, "tcp client"),
	}
}

func (c *TCPClient) handleConnection(ctx context.Context, conn net.Conn) {
}

func (c *TCPClient) Close() {

}

func (c *TCPClient) sendServer(encoder *gob.Encoder, dto string) error {
	if err := encoder.Encode(&dto); err != nil {
		fmt.Println("Error encoding DTO1:", err)
		return err
	}
	return nil
}

func (c *TCPClient) Start(ctx context.Context, ch chan<- MysqlOperation, gtidset string) {
	c.logger.Info("start " + c.ServerAddress)
	conn, err := net.Dial("tcp", c.ServerAddress)
	if err != nil {
		c.logger.Error("connection error: " + err.Error())
		return
	}
	defer conn.Close()

	zstdReader, err := zstd.NewReader(conn)
	if err != nil {
		c.logger.Error("Error creating zstd reader:" + err.Error())
		return
	}
	defer zstdReader.Close()

	decoder := gob.NewDecoder(zstdReader)
	encoder := gob.NewEncoder(conn)

	if err := c.sendServer(encoder, fmt.Sprintf("gtidset@%s", gtidset)); err != nil {
		c.logger.Error("sender gtidset faile " + err.Error())
		return
	}

	var cacheMysqlOperations []MysqlOperation = []MysqlOperation{}

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			var operations []MysqlOperation
			if err := decoder.Decode(&operations); err != nil {
				c.logger.Error("Error decoding message:" + err.Error())
				break loop
			}

			for _, oper := range operations {
				select {
				case ch <- oper:
				default:
					c.logger.Info(fmt.Sprintf("Cache MysqlOperation %v", oper))
					cacheMysqlOperations = append(cacheMysqlOperations, oper)
				}
			}

			if len(cacheMysqlOperations) >= 1 {
				c.sendServer(encoder, "pause")
				c.logger.Info("sender to server 'pause'.")

				c.logger.Info(fmt.Sprintf("Cache MysqlOperation number: %d", len(cacheMysqlOperations)))

				for len(cacheMysqlOperations) > 0 {
					select {
					case ch <- cacheMysqlOperations[0]:
						cacheMysqlOperations = cacheMysqlOperations[1:]
					}
				}
				if len(cacheMysqlOperations) == 0 {
					c.sendServer(encoder, "resume")
					c.logger.Info("sender to server 'resume'.")
				}
			}
		}
	}
	c.logger.Info("stopped.")
}
