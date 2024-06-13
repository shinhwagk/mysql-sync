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
	Logger *Logger

	listenAddress string

	Clients sync.Map

	// metric   *MetricTCPServer
	moCh      <-chan MysqlOperation
	gtidsetCh chan<- string
	metricCh  chan<- interface{}
}

type TcpServerClient struct {
	conn    net.Conn
	channel chan MysqlOperation
	close   chan bool
}

type SendOperationControl struct {
	ReceiveCountCh chan uint
}

func NewTCPServer(logLevel int, address string, moCh <-chan MysqlOperation, gtidsetCh chan<- string, metricCh chan<- interface{}) *TCPServer {
	return &TCPServer{
		Logger:        NewLogger(logLevel, "tcp server"),
		listenAddress: address,
		metricCh:      metricCh,
		gtidsetCh:     gtidsetCh,
		moCh:          moCh,
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
		timer := time.NewTimer(time.Second * 1)
		defer timer.Stop()
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
				timer.Reset(time.Second * 1)
			case <-timer.C:
				timer.Reset(time.Second * 1)
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

	timer := time.NewTimer(time.Second * 1)
	defer timer.Stop()

	resumeTimer := time.NewTimer(100 * time.Millisecond)
	defer resumeTimer.Stop()

	var mysqlOperations []MysqlOperation

	var count = 0

	var dida = 0
	for {
		fmt.Println("111", dida)
		dida += 1
		select {
		case <-ctx.Done():
			s.Logger.Info("ctx done signal received.")
			return
		case <-timer.C:
			timer.Reset(time.Second * 1)
		case rc, ok := <-rcCh:
			fmt.Println("Rc", rc)
			if !ok {
				s.Logger.Info("ReceiveCountCh")
				return
			}
			var ccount = 0
			for rc >= 1 {
				rcCnt := rc

				if rc >= 10 {
					rcCnt = 10
				}

				select {
				case oper, ok := <-tsc.channel:
					count += 1

					if !ok {
						s.Logger.Info("mysql operation channel closed.")
						return
					}

					mysqlOperations = append(mysqlOperations, oper)

					fmt.Println("rc sendcnt,", rc, rcCnt)

					if len(mysqlOperations) >= rcCnt {
						sendMO := mysqlOperations[:rcCnt]
						mysqlOperations = mysqlOperations[rcCnt:]
						if err := s.sendClient(encoder, zstdWriter, sendMO); err != nil {
							s.Logger.Error("sendClient error: " + err.Error())
							return
						}
						rc -= rcCnt

						ccount += rcCnt
						fmt.Println("ccount ccount", ccount, rc)
						s.Logger.Debug(fmt.Sprintf("Compressed data sent: %d bytes\n", buf.Len()))
						s.metricCh <- MetricUnit{Name: MetricTCPServerOperations, Value: uint(len(sendMO))}
					}

					resumeTimer.Reset(time.Millisecond * 100)
				case <-resumeTimer.C:
					fmt.Println("rc sendcnt,", rc, rcCnt)

					if len(mysqlOperations) >= rcCnt {
						sendMO := mysqlOperations[:rcCnt]
						mysqlOperations = mysqlOperations[rcCnt:]
						if err := s.sendClient(encoder, zstdWriter, sendMO); err != nil {
							s.Logger.Error("sendClient error: " + err.Error())
							return
						}
						rc -= rcCnt

						ccount += rcCnt
						fmt.Println("ccount ccount", ccount, rc)
						s.Logger.Debug(fmt.Sprintf("Compressed data sent: %d bytes\n", buf.Len()))
						s.metricCh <- MetricUnit{Name: MetricTCPServerOperations, Value: uint(len(sendMO))}
					}

					// if len(mysqlOperations) > 0 {
					// 	ccount += len(mysqlOperations)

					// 	fmt.Println("send operations1", len(mysqlOperations), ccount)

					// 	if err := s.sendClient(encoder, zstdWriter, mysqlOperations); err != nil {
					// 		s.Logger.Error("sendClient error: " + err.Error())
					// 		return
					// 	}
					// 	// s.logger.Debug(fmt.Sprintf("Compressed data sent: %d bytes\n", buf.Len()))
					// 	s.Logger.Debug(fmt.Sprintf("timeout 100 millisecond send operation %d", len(mysqlOperations)))
					// 	mysqlOperations = mysqlOperations[:0]
					// 	s.metricCh <- MetricUnit{Name: MetricTCPServerOperations, Value: uint(len(mysqlOperations))}

					// }
					resumeTimer.Reset(time.Millisecond * 100)
				}
			}
			timer.Reset(time.Second * 1)
		}
	}
}

func (s *TCPServer) handleFromClient(ctx context.Context, tsc *TcpServerClient, rcCh chan<- int) {
	decoder := gob.NewDecoder(tsc.conn)

	for {
		select {
		case <-ctx.Done():
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

			if strings.HasPrefix(signal, "gtidset@") {
				s.Logger.Info("from client resdf gtidset " + signal)
				parts := strings.Split(signal, "@")
				s.gtidsetCh <- parts[1]
			} else if strings.HasPrefix(signal, "receive@") {
				parts := strings.Split(signal, "@")
				if len(parts) == 2 {
					result, err := strconv.Atoi(parts[1])
					if err != nil {
						fmt.Println("Error converting string to int:", err)
					} else {
						fmt.Println("Converted integer1:", result)
						rcCh <- result
						fmt.Println("Converted integer:", result)
					}
				}
			} else {
				fmt.Println("dcode nill")
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
