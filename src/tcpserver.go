package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

func init() {
	gob.Register(MysqlOperationHeartbeat{})
	gob.Register(MysqlOperationDDLDatabase{})
	gob.Register(MysqlOperationGTID{})

	// gob.Register(&DTOPause{})
	// gob.Register(&DTOResume{})
	// gob.Register(&DTOGtidSet{})

}

type TCPServer struct {
	logger *Logger

	listenAddress string

	Clients sync.Map

	// ch <-chan MysqlOperation
}

type Client struct {
	conn    net.Conn
	channel chan MysqlOperation
}

type ServerConnectionState struct {
	paused bool
}

func NewTCPServer(logLevel int, address string) *TCPServer {
	return &TCPServer{
		logger:        NewLogger(logLevel, "tcp server"),
		listenAddress: address,
	}
}

func (s *TCPServer) Start(ctx context.Context, moCh <-chan MysqlOperation, gtidsetsh chan<- string) error {
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Error listening: " + err.Error()))
	}
	defer listener.Close()
	s.logger.Info("listening on:" + s.listenAddress)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		listener.Close()
	}()

	go func() {
		for msg := range moCh {
			s.Clients.Range(func(key, value interface{}) bool {
				client := value.(*Client)
				client.channel <- msg
				return true
			})
		}
	}()

	for {
		conn, err := listener.Accept()
		s.logger.Info("start listener...")

		if err != nil {
			s.logger.Info("Error accepting:" + err.Error())
			return err
		}

		clientChannel := make(chan MysqlOperation, 100)
		client := &Client{
			conn:    conn,
			channel: clientChannel,
		}

		s.Clients.Store(conn.RemoteAddr().String(), client)

		go s.handleConnection(client, gtidsetsh)
	}

}

func (s *TCPServer) handleConnection(client *Client, gtidsetCh chan<- string) {
	defer func() {
		client.conn.Close()
		s.Clients.Delete(client.conn.RemoteAddr().String())
		close(client.channel)
	}()

	resumeChan := make(chan struct{})

	var wg sync.WaitGroup

	wg.Add(2)

	var connState = &ServerConnectionState{paused: false}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer wg.Done()
		s.handleFromClient(ctx, client.conn, connState, gtidsetCh, resumeChan)
		s.logger.Info("tcp from client handler close.")
		cancel()
	}()

	go func() {
		defer wg.Done()
		s.handleToClient(ctx, client.conn, connState, client.channel, gtidsetCh, resumeChan)
		s.logger.Info("tcp to client handler close.")
		cancel()
	}()

	wg.Wait()

	s.logger.Info(fmt.Sprintf("tcp connection '%s' close.", client.conn.RemoteAddr().String()))
}

func (s *TCPServer) handleToClient(ctx context.Context, conn net.Conn, scs *ServerConnectionState, ch <-chan MysqlOperation, gtidsetsh chan<- string, resumeChan <-chan struct{}) {
	var buf bytes.Buffer

	multiWriter := io.MultiWriter(conn, &buf)
	zstdWriter, err := zstd.NewWriter(multiWriter)

	if err != nil {
		fmt.Println("Error creating zstd writer:", err)
		return
	}
	defer zstdWriter.Close()

	encoder := gob.NewEncoder(zstdWriter)

	timer := time.NewTimer(100 * time.Millisecond)

	var mysqlOperations []MysqlOperation

	for {
		s.logger.Debug("sendClient error:  + err.Error()")

		if scs.paused {
			select {
			case <-resumeChan:
				fmt.Println("Resuming event processing...")
			case <-time.After(time.Second * 10):
				fmt.Println("Timeout waiting for resume, checking pause status again...")
				continue
			}
		}

		select {
		case <-ctx.Done():
			return
		case val := <-ch:
			mysqlOperations = append(mysqlOperations, val)
			s.logger.Info("received value")

			if len(mysqlOperations) >= 10 {
				if err := s.sendClient(encoder, zstdWriter, mysqlOperations); err != nil {
					s.logger.Error("sendClient error: " + err.Error())
					return
				}
				fmt.Printf("Compressed data sent: %d bytes\n", buf.Len())
				mysqlOperations = mysqlOperations[:0]
				timer.Reset(100 * time.Millisecond)
			}

		case <-timer.C:
			if len(mysqlOperations) > 0 {
				if err := s.sendClient(encoder, zstdWriter, mysqlOperations); err != nil {
					s.logger.Error("sendClient error: " + err.Error())
					return
				}
				fmt.Printf("Compressed data sent: %d bytes\n", buf.Len())
				mysqlOperations = mysqlOperations[:0]
			}
			timer.Reset(100 * time.Millisecond)
		default:
			time.Sleep(1 * time.Second)
		}
		buf.Reset()
	}
}

func (s *TCPServer) handleFromClient(ctx context.Context, conn net.Conn, scs *ServerConnectionState, gtidsetCh chan<- string, resumeChan chan struct{}) {
	decoder := gob.NewDecoder(conn)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var dto string
			if err := decoder.Decode(&dto); err != nil {
				if err == io.EOF {
					s.logger.Info("tcp client close.")
				} else {
					s.logger.Error("Error decoding message:" + err.Error())
				}
				return
			}

			fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")

			if strings.HasPrefix(dto, "gtidset") {
				fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1")

				parts := strings.Split(dto, "@")
				gtidsetCh <- parts[1]
				fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1e")

			} else if dto == "pause" {
				fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx2")

				fmt.Printf("Received DTOPause: %+v\n", dto)
				if scs.paused {
					select {
					case <-resumeChan:
						fmt.Println("Resuming event processing...")
					}
				}
				fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx2e")

			} else if dto == "resume" {
				fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx3")

				scs.paused = false
				resumeChan <- struct{}{}
				fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx3e")

			} else {
				fmt.Println("dcode nill")
			}

			fmt.Println("yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy")

			// switch v := dto.(type) {
			// case *DTOPause:
			// 	fmt.Printf("Received DTOPause: %+v\n", v)
			// 	s.paused = true
			// case *DTOResume:
			// 	fmt.Printf("Received DTO2: %+v\n", v)
			// 	s.paused = false
			// 	resumeChan <- struct{}{}
			// case *DTOGtidSet:
			// 	gtidsetChan <- v.GtidSet
			// 	fmt.Printf("Received DTO2: %+v\n", v)
			// default:
			// 	fmt.Printf("Received unknown type: %+v\n", v)
			// }
		}
	}
}

func (s *TCPServer) sendClient(encoder *gob.Encoder, zstdWriter *zstd.Encoder, operations []MysqlOperation) error {
	if err := encoder.Encode(operations); err != nil {
		fmt.Println("Error encoding message:", err)
		return err
	}

	if err := zstdWriter.Flush(); err != nil {
		fmt.Println("Error flushing zstd writer:", err)
		return err
	}
	return nil
}
