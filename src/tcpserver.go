package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log"
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
	close   chan bool
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
		log.Println("Shutting down server...")
		listener.Close()
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case mo, ok := <-s.moCh:
				if !ok {
					s.Logger.Info("mysql operation channal close.")
					return
				}
				s.Clients.Range(func(key, value interface{}) bool {
					client := value.(*TcpServerClient)
					client.channel <- mo
					return true
				})
			case <-time.After(time.Second * 1):
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

		clientChannel := make(chan MysqlOperation, 100)
		client := &TcpServerClient{
			conn:    conn,
			channel: clientChannel,
		}

		s.Clients.Store(conn.RemoteAddr().String(), client)

		go s.handleConnection(client)
	}

}

func (s *TCPServer) handleConnection(client *TcpServerClient) {
	defer func() {
		client.conn.Close()
		close(client.channel)
		s.Clients.Delete(client.conn.RemoteAddr().String())
	}()

	var wg sync.WaitGroup

	wg.Add(2)

	// soc := &SendOperationControl{ReceiveCountCh: make(chan uint)}

	rcCh := make(chan int)
	// go func() {
	// 	rcCh <- 100
	// }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer wg.Done()
		s.handleFromClient(ctx, client, rcCh)
		s.Logger.Info(fmt.Sprintf("tcp from client(%s) handler close.", client.conn.RemoteAddr().String()))
		cancel()
	}()

	go func() {
		defer wg.Done()
		s.handleToClient(ctx, client, rcCh)
		s.Logger.Info(fmt.Sprintf("tcp to client(%s) handler close.", client.conn.RemoteAddr().String()))
		cancel()
	}()

	wg.Wait()

	s.Logger.Info(fmt.Sprintf("tcp connection '%s' close.", client.conn.RemoteAddr().String()))
}

func (s *TCPServer) handleToClient(ctx context.Context, tsc *TcpServerClient, rcCh <-chan int) {
	var buf bytes.Buffer

	multiWriter := io.MultiWriter(tsc.conn, &buf)
	zstdWriter, err := zstd.NewWriter(multiWriter)

	if err != nil {
		fmt.Println("Error creating zstd writer:", err)
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
				case oper, ok := <-tsc.channel:
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

						s.Logger.Debug(fmt.Sprintf("Compressed data sent: %d bytes\n", buf.Len()))
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

						s.Logger.Debug(fmt.Sprintf("Compressed data sent: %d bytes\n", buf.Len()))
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
		select {
		case <-ctx.Done():
			s.Logger.Info("ctx done signal received.")
			return
		default:
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
				s.Logger.Info("from client resdf gtidsets " + signal)
				parts := strings.Split(signal, "@")
				s.gsCh <- parts[1]
			} else if strings.HasPrefix(signal, "receive@") {
				parts := strings.Split(signal, "@")
				if len(parts) == 2 {
					result, err := strconv.Atoi(parts[1])
					if err != nil {
						s.Logger.Error(fmt.Sprintf("Converting signal '%s' error: %s", signal, err.Error()))
					} else {
						rcCh <- result
					}
				}
			} else {
				s.Logger.Error(fmt.Sprintf("Unknow signal '%s'.", signal))
			}
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

	s.Logger.Debug(fmt.Sprintf("send operations number:'%d' to client success.", len(operations)))
	return nil
}
