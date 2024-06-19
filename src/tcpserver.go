package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

func init() {
	gob.Register(MysqlOperationHeartbeat{})
	gob.Register(MysqlOperationDDLDatabase{})
	gob.Register(MysqlOperationGTID{})
	gob.Register(MysqlOperationDDLTable{})
	gob.Register(MysqlOperationDMLInsert{})
	gob.Register(MysqlOperationDMLUpdate{})
	gob.Register(MysqlOperationDMLDelete{})
	gob.Register(MysqlOperationBegin{})
	gob.Register(MysqlOperationXid{})
}

type TCPServer struct {
	Logger        *Logger
	listenAddress string

	gsCh     chan<- string
	moCh     <-chan MysqlOperation
	metricCh chan<- MetricUnit

	Clients sync.Map
}

type TcpServerClient struct {
	conn    net.Conn
	channel chan MysqlOperation
	close   bool
}

type SendOperationControl struct {
	ReceiveCountCh chan uint
}

func NewTCPServer(logLevel int, listenAddress string, gsCh chan<- string, moCh <-chan MysqlOperation, metricCh chan<- MetricUnit) *TCPServer {
	return &TCPServer{
		Logger:        NewLogger(logLevel, "tcp server"),
		listenAddress: listenAddress,

		gsCh:     gsCh,
		moCh:     moCh,
		metricCh: metricCh,
	}
}

func (s *TCPServer) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Error listening: " + err.Error()))
	}
	defer listener.Close()
	s.Logger.Info("listening on:" + s.listenAddress)

	go func() {
		<-ctx.Done()
		s.Logger.Info("Shutting down server...")
		listener.Close()
	}()

	go func() {
		for {
			s.Logger.Debug("Distribute mysql operation to clients.")
			select {
			case <-time.After(time.Second * 1):
			case <-ctx.Done():
				return
			case mo, ok := <-s.moCh:
				if !ok {
					s.Logger.Info("mysql operation channal close.")
					return
				}
				fmt.Println("shoudao mo")
				s.Clients.Range(func(key, value interface{}) bool {
					client := value.(*TcpServerClient)
					fmt.Println("send cleint", client.conn.RemoteAddr().String())
					fmt.Println("send clein", client.close)
					if client.close {
						s.Clients.Delete(client.conn.RemoteAddr().String())
						select {
						case <-client.channel:
						case <-time.After(time.Second):
						}
						// close(client.channel)

					} else {
						fmt.Println("ssssssssss", mo, client.channel, &client.channel)
						select {
						case client.channel <- mo:
						case <-time.After(time.Second * 5):
							fmt.Println("发送操作超时")
						}
						fmt.Println("fffffffffffffff")

					}
					fmt.Println("send clein2t", client.conn.RemoteAddr().String())

					return true
				})
				fmt.Println("shoudao mo1")
			}
		}
	}()

	for {
		conn, err := listener.Accept()
		s.Logger.Info("start listener...")

		if err != nil {
			s.Logger.Info("Error accepting:" + err.Error())
			return err
		}

		clientChannel := make(chan MysqlOperation)
		client := &TcpServerClient{
			conn:    conn,
			channel: clientChannel,
			close:   false,
		}

		s.Logger.Info("Add client(%s) to clients.", conn.RemoteAddr().String())

		s.Clients.Store(conn.RemoteAddr().String(), client)

		go s.handleConnection(client)
	}

}

func (s *TCPServer) handleConnection(client *TcpServerClient) {
	s.Logger.Info("Receive new client(%s) ", client.conn.RemoteAddr().String())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		client.close = true
		client.conn.Close()
	}()

	rcCh := make(chan int)
	defer close(rcCh)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		s.handleFromClient(ctx, client, rcCh)
		s.Logger.Info("tcp from client(%s) handler close.", client.conn.RemoteAddr().String())
		cancel()
	}()

	go func() {
		defer wg.Done()
		s.handleToClient(ctx, client, rcCh)
		s.Logger.Info("tcp to client(%s) handler close.", client.conn.RemoteAddr().String())
		cancel()
	}()

	wg.Wait()

	s.Logger.Info("tcp connection '%s' close.", client.conn.RemoteAddr().String())
}

func (s *TCPServer) handleToClient(ctx context.Context, client *TcpServerClient, rcCh <-chan int) {
	var buf bytes.Buffer

	multiWriter := io.MultiWriter(client.conn, &buf)
	zstdWriter, err := zstd.NewWriter(multiWriter)

	if err != nil {
		s.Logger.Error("Error creating zstd writer: %s", err.Error())
		return
	}
	defer zstdWriter.Close()

	encoder := gob.NewEncoder(zstdWriter)

	var mysqlOperations []MysqlOperation

	for {
		select {
		case <-time.After(time.Second * 1):
		case <-ctx.Done():
			s.Logger.Info("ctx done signal received.")
			return
		case rc, ok := <-rcCh:
			if !ok {
				s.Logger.Info("ReceiveCountCh")
				return
			}

			for rc >= 1 {
				rcCnt := rc

				if rc >= 10 {
					rcCnt = 10
				}

				select {
				case <-ctx.Done():
					s.Logger.Info("ctx done signal received.")
					return
				case oper, ok := <-client.channel:
					if !ok {
						s.Logger.Info("mysql operation channel closed.")
						return
					}

					mysqlOperations = append(mysqlOperations, oper)

					if len(mysqlOperations) >= rcCnt {
						sendMO := mysqlOperations[:rcCnt]
						mysqlOperations = mysqlOperations[rcCnt:]
						if err := s.sendClient(encoder, zstdWriter, sendMO); err != nil {
							s.Logger.Error("sendClient error: " + err.Error())
							return
						}
						rc -= rcCnt

						// s.Logger.Debug(fmt.Sprintf("Compressed data sent: %d bytes\n", buf.Len()))
						s.metricCh <- MetricUnit{Name: MetricTCPServerOutgoing, Value: uint(buf.Len())}
						s.metricCh <- MetricUnit{Name: MetricTCPServerSendOperations, Value: uint(len(sendMO))}
						buf.Reset()
					}
				case <-time.After((time.Millisecond * 100)):
					if len(mysqlOperations) >= rcCnt {
						sendMO := mysqlOperations[:rcCnt]
						mysqlOperations = mysqlOperations[rcCnt:]
						if err := s.sendClient(encoder, zstdWriter, sendMO); err != nil {
							s.Logger.Error("sendClient error: " + err.Error())
							return
						}
						rc -= rcCnt

						// s.Logger.Debug(fmt.Sprintf("Compressed data sent: %d bytes\n", buf.Len()))
						s.metricCh <- MetricUnit{Name: MetricTCPServerOutgoing, Value: uint(buf.Len())}
						s.metricCh <- MetricUnit{Name: MetricTCPServerSendOperations, Value: uint(len(sendMO))}
						buf.Reset()
					}
				}
			}
		}
	}
}

func (s *TCPServer) handleFromClient(ctx context.Context, tsc *TcpServerClient, rcCh chan<- int) {
	decoder := gob.NewDecoder(tsc.conn)

	for {
		var signal string

		if err := decoder.Decode(&signal); err != nil {
			if err == io.EOF {
				s.Logger.Info("tcp client close.")
			} else {
				s.Logger.Error("Error decoding message:" + err.Error())
			}
			return
		}

		if strings.HasPrefix(signal, "gtidsets@") {
			s.Logger.Info("from client receive gtidsets " + signal)
			parts := strings.Split(signal, "@")
			s.gsCh <- parts[1]
		} else if strings.HasPrefix(signal, "receive@") {
			parts := strings.Split(signal, "@")
			if len(parts) == 2 {
				result, err := strconv.Atoi(parts[1])
				if err != nil {
					s.Logger.Error("Converting signal '%s' error: %s", signal, err.Error())
				} else {
					rcCh <- result
				}
			}
		} else {
			s.Logger.Error("Unknow signal '%s'.", signal)
		}
	}
}

func (s *TCPServer) sendClient(encoder *gob.Encoder, zstdWriter *zstd.Encoder, operations []MysqlOperation) error {
	if err := encoder.Encode(operations); err != nil {
		s.Logger.Error("Error encoding message: %s", err.Error())
		return err
	}

	if err := zstdWriter.Flush(); err != nil {
		s.Logger.Error("Error flushing zstd writer: %s", err)
		return err
	}

	s.Logger.Debug("Mysql operations number:'%d' sent to client successfully.", len(operations))
	return nil
}
