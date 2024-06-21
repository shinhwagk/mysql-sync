package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
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

func (ts *TCPServer) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", ts.listenAddress)
	if err != nil {
		ts.Logger.Error(fmt.Sprintf("Error listening: " + err.Error()))
	}
	defer listener.Close()
	ts.Logger.Info("listening on:" + ts.listenAddress)

	go func() {
		<-ctx.Done()
		ts.Logger.Info("Shutting down server...")
		listener.Close()
	}()

	go func() {
		for {
			select {
			case <-time.After(time.Second * 1):
			case <-ctx.Done():
				ts.Logger.Info("Context canceled, stopping distribution loop.")
				return
			case mo, ok := <-ts.moCh:
				if !ok {
					ts.Logger.Info("mysql operation channal close.")
					return
				}

				hasClients := false

				ts.Clients.Range(func(key, value interface{}) bool {
					hasClients = true
					client := value.(*TcpServerClient)
					if client.close {
						ts.Clients.Delete(client.conn.RemoteAddr().String())
						client.SetClose()
						ts.Logger.Info("Remove client(%s) from clients.", client.conn.RemoteAddr().String())
					} else {
						tryCnt := 5
						for i := 0; i < tryCnt; i++ {
							select {
							case <-client.ctx.Done():
								i = tryCnt
							case client.channel <- mo:
								ts.Logger.Debug(fmt.Sprintf("moCh -> mo -> client cache(%s) ok.", client.conn.RemoteAddr().String()))
								i = tryCnt
							case <-time.After(time.Second * 1):
								fmt.Println("mo -> client mo timeout.", len(client.channel))
								// if i == tryCnt-1 {
								// 	client.cancel()
								// 	i = tryCnt
								// }
							}
						}
					}
					return true
				})

				if !hasClients {
					ts.Logger.Debug("No clients, mod discard.")
					time.Sleep(time.Second * 1)
					continue
				}
			}
		}
	}()

	for {
		conn, err := listener.Accept()
		ts.Logger.Info("Listener started.")

		if err != nil {
			ts.Logger.Info("Error accepting:" + err.Error())
			return err
		}

		if client, err := NewTcpServerClient(ts.Logger.Level, conn); err != nil {
			ts.Logger.Error("Create Client(%s) error: %s", conn.RemoteAddr().String(), err.Error())
			return err
		} else {
			ts.Clients.Store(conn.RemoteAddr().String(), client)
			ts.Logger.Info("Add client(%s) to clients.", conn.RemoteAddr().String())
			go ts.handleConnection(client)
		}
	}
}

func (ts *TCPServer) handleConnection(tsClient *TcpServerClient) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		ts.handleFromClient(tsClient)
		ts.Logger.Info("tcp from client(%s) handler close.", tsClient.conn.RemoteAddr().String())
		tsClient.SetClose()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ts.handleToClient(tsClient)
		ts.Logger.Info("tcp to client(%s) handler close.", tsClient.conn.RemoteAddr().String())
		tsClient.SetClose()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tsClient.Cleanup(); err != nil {
			ts.Logger.Error(fmt.Sprintf("Close connection error: " + err.Error()))
		}
	}()

	ts.Logger.Info("Receive new client(%s) ", tsClient.conn.RemoteAddr().String())
	wg.Wait()

	ts.Logger.Info("Client(%s) closed.", tsClient.conn.RemoteAddr().String())
}

func (s *TCPServer) handleToClient(tsClient *TcpServerClient) {
	var mysqlOperations []MysqlOperation

	for {
		select {
		case <-time.After(time.Second * 1):
			s.Logger.Debug("wait receive singnal")
		case <-tsClient.ctx.Done():
			s.Logger.Info("client ctx done signal received.")
			return
		case rc, ok := <-tsClient.rcCh:
			if !ok {
				s.Logger.Info("ReceiveCountCh")
				return
			}

			for rc >= 1 {
				rcCnt := rc

				if rc >= 1000 {
					rcCnt = 100
				} else if rc >= 100 {
					rcCnt = 10
				}

				select {
				case <-tsClient.ctx.Done():
					s.Logger.Info("ctx done signal received.")
					return
				case oper, ok := <-tsClient.channel:
					if !ok {
						s.Logger.Info("mysql operation channel closed.")
						return
					}

					mysqlOperations = append(mysqlOperations, oper)

					if len(mysqlOperations) >= rcCnt {
						mos := mysqlOperations[:rcCnt]
						mysqlOperations = mysqlOperations[rcCnt:]
						if err := s.sendClient(tsClient, mos); err != nil {
							s.Logger.Error("sendClient error: " + err.Error())
							return
						}
						rc -= rcCnt

						s.metricCh <- MetricUnit{Name: MetricTCPServerOutgoing, Value: uint(tsClient.encoderBuffer.Len())}
						s.metricCh <- MetricUnit{Name: MetricTCPServerSendOperations, Value: uint(len(mos))}
						tsClient.encoderBuffer.Reset()
					}
				case <-time.After((time.Millisecond * 100)):
					if len(mysqlOperations) >= rcCnt {
						mos := mysqlOperations[:rcCnt]
						mysqlOperations = mysqlOperations[rcCnt:]
						if err := s.sendClient(tsClient, mos); err != nil {
							s.Logger.Error("sendClient error: " + err.Error())
							return
						}
						rc -= rcCnt

						// s.Logger.Debug(fmt.Sprintf("Compressed data sent: %d bytes\n", buf.Len()))
						s.metricCh <- MetricUnit{Name: MetricTCPServerOutgoing, Value: uint(tsClient.encoderBuffer.Len())}
						s.metricCh <- MetricUnit{Name: MetricTCPServerSendOperations, Value: uint(len(mos))}
						tsClient.encoderBuffer.Reset()
					}
				}
			}
		}
	}
}

func (s *TCPServer) handleFromClient(tsClient *TcpServerClient) {
	for {
		var signal string

		if err := tsClient.decoder.Decode(&signal); err != nil {
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
					tsClient.rcCh <- result
				}
			}
		} else {
			s.Logger.Error("Unknow signal '%s'.", signal)
		}
	}
}

func (s *TCPServer) sendClient(tsClient *TcpServerClient, mos []MysqlOperation) error {
	if err := tsClient.encoder.Encode(mos); err != nil {
		s.Logger.Error("Error encoding message: %s", err.Error())
		return err
	}

	if err := tsClient.encoderZstdWriter.Flush(); err != nil {
		s.Logger.Error("Error flushing zstd writer: %s", err)
		return err
	}

	s.Logger.Debug("moCh -> mo -> client cache -> mo(%d) -> client ok.", len(mos))
	return nil
}
