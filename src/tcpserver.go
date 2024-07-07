package main

import (
	"bufio"
	"context"
	"encoding/gob"
	"fmt"
	"net"
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

	gob.Register(Signal1{})
	gob.Register(Signal2{})
}

type TCPServer struct {
	Logger        *Logger
	listenAddress string

	ctx    context.Context
	cancel context.CancelFunc

	moCh     <-chan MysqlOperation
	metricCh chan<- MetricUnit

	Clients map[string]*TCPServerClient

	BatchID uint
}

func NewTCPServer(logLevel int, listenAddress string, clientNames []string, moCh <-chan MysqlOperation, metricCh chan<- MetricUnit) *TCPServer {
	clients := make(map[string]*TCPServerClient)

	for _, name := range clientNames {
		clients[name] = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &TCPServer{
		Logger:        NewLogger(logLevel, "tcp server"),
		listenAddress: listenAddress,

		ctx:    ctx,
		cancel: cancel,

		moCh:     moCh,
		metricCh: metricCh,

		Clients: clients,
		BatchID: 0,
	}
}

func (ts *TCPServer) Start(ctx context.Context) {
	ts.Logger.Info("Started.")
	defer ts.Logger.Info("Closed.")

	listener, err := net.Listen("tcp", ts.listenAddress)
	if err != nil {
		ts.Logger.Error("Listening: %s", err.Error())

	}

	go func() {
		<-ctx.Done()
		ts.cancel()
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ts.ctx.Done()
		// clear clients
		for _, client := range ts.Clients {
			if client != nil {
				client.cancel()
			}
		}
		// close listener
		if err := listener.Close(); err != nil {
			ts.Logger.Error("Listener Close: %s", err.Error())
		} else {
			ts.Logger.Info("Listener Closed.")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ts.distributor()
		ts.cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ts.handleClients(listener)
		ts.cancel()
	}()

	wg.Wait()
}

func (ts *TCPServer) clientsReady() bool {
	for name, client := range ts.Clients {
		if client == nil {
			return false
		}
		ts.Logger.Debug("Clients status client(%s), dead: %t, state: %s", client.Name, client.Dead, client.State)
		if client.Dead {
			return false
		}

		if client.State == ClientBusy {
			if client.SendError != nil {
				ts.Logger.Info("Push to Client(%s) Batch(%d) again.", name, client.SendBatchID)
				client.PushClient()
			}
			return false
		}

		if client.State != ClientFree {
			return false
		}
	}
	return true
}

func (ts *TCPServer) distributor() {
	ts.Logger.Info("Distributor started.")
	defer ts.Logger.Info("Distributor closed.")

	maxRcCnt := cap(ts.moCh)
	const minRcCnt int = 10

	sendTimestamp := time.Now()

	noReadyMs := 0

	fetchDelayMs := 0
	sendBaseLineDelayMs := 0
	sendBaseLineMaxCount := 0
	fetchCount := 0
	resetBaseLine := true

	// min delay 1ms
	delayMsSlice := make([]int, 10) // maxSize must greater then 3

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			if !ts.clientsReady() {
				if noReadyMs <= 1000 {
					noReadyMs += 10
				}
				ts.Logger.Debug("Clients not ready sleep: %d.", noReadyMs)
				time.Sleep(time.Millisecond * time.Duration(noReadyMs))
				continue
			}
			noReadyMs = 0

			ts.Logger.Debug("MoCh cache capacity: %d/%d.", len(ts.moCh), maxRcCnt)
			sendDelayMs := max(int(time.Since(sendTimestamp).Milliseconds()), 1) // min delay 1ms

			if fetchCount >= minRcCnt {
				if resetBaseLine {
					sendBaseLineDelayMs = int(float64(sendDelayMs) * 1.2)
					resetBaseLine = false
					fetchCount = max(sendBaseLineMaxCount, fetchCount)
					fmt.Println("xxxxxxx", sendDelayMs, fetchCount, sendBaseLineDelayMs, fetchCount, sendBaseLineMaxCount)
				} else {
					delayMsSlice = updateSlice(delayMsSlice, sendDelayMs)

					select {
					case <-ticker.C:
						fetchCount = minRcCnt
						resetBaseLine = true
					default:
						avgSendDelayMs := calculateAdjustedMean(delayMsSlice)
						fmt.Println("xxxxxxx avgSendDelayMs:", avgSendDelayMs, sendBaseLineDelayMs)

						if avgSendDelayMs <= sendBaseLineDelayMs {
							sendBaseLineMaxCount = max(fetchCount, sendBaseLineMaxCount)
							fetchCount += minRcCnt
						} else {
							sendBaseLineMaxCount = max(sendBaseLineMaxCount-minRcCnt, minRcCnt) // not less than minRcCnt
							fetchCount = max(fetchCount-minRcCnt, minRcCnt)                     // not less than minRcCnt
						}
						fmt.Println("xxxxxxx max(fetchCount, sendBaseLineMaxCount)", fetchCount, sendBaseLineMaxCount)
						fetchCount = max(fetchCount, sendBaseLineMaxCount)
					}
				}
			}

			fetchCount = min(fetchCount, len(ts.moCh))
			// multiples of minRcCnt
			fetchCount = fetchCount / minRcCnt * minRcCnt
			// Ensure fetchCount is never less than minRcCnt. This line is critical and must remain here for correct functionality.
			fetchCount = max(fetchCount, minRcCnt)
			fmt.Println("xxxxxxxf", fetchCount, sendDelayMs, fetchDelayMs, sendBaseLineDelayMs, resetBaseLine, sendBaseLineMaxCount, len(ts.moCh))

			fetchTimestamp := time.Now()
			if mos, err := ts.fetchMos(fetchCount); err != nil {
				ts.Logger.Error("Fetch mos fetch count: %d err: %s", fetchCount, err.Error())
				return
			} else {
				fetchDelayMs = int(time.Since(fetchTimestamp).Milliseconds())
				ts.BatchID += 1
				ts.Logger.Info("Push batch(%d) mos(%d) to clients.", ts.BatchID, len(mos))
				sendTimestamp = time.Now()
				ts.pushClients(mos)
			}
		}
	}
}

func (ts *TCPServer) pushClients(mos []MysqlOperation) {
	var wg sync.WaitGroup
	for name, client := range ts.Clients {
		wg.Add(1)
		go func(cli *TCPServerClient) {
			defer wg.Done()
			cli.SetPush(ts.BatchID, mos)
			ts.Logger.Info("Push to Client(%s) Batch(%d).", name, ts.BatchID)
			cli.PushClient()
		}(client)
	}
	wg.Wait()
}

func (ts *TCPServer) fetchMos(count int) ([]MysqlOperation, error) {
	mos := make([]MysqlOperation, 0, count)
	for i := 0; i < count; i++ {
		op, ok := <-ts.moCh
		if !ok {
			return nil, fmt.Errorf("channel closed during reception")
		}
		mos = append(mos, op)
	}
	return mos, nil
}

func (ts *TCPServer) handleClients(listener net.Listener) {
	ts.Logger.Info("HandleClients started.")
	defer ts.Logger.Info("HandleClients closed.")

	var wg sync.WaitGroup
	for {
		conn, err := listener.Accept()
		if err != nil {
			ts.Logger.Error("Accepting: %s", err.Error())
			break
		}

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				ts.Logger.Error("Failed to read from client: %s", err.Error())
			} else {
				ts.Logger.Error("Client disconnected before sending a name")

			}
		}

		clientName := scanner.Text()

		if client, exists := ts.Clients[clientName]; exists {
			if client == nil {
				if ts.Clients[clientName], err = NewTcpServerClient(ts.Logger.Level, clientName, ts.metricCh, conn); err != nil {
					ts.Logger.Error("Client(%s) Init: %s", clientName, err.Error())
					return
				} else {
					wg.Add(1)
					go func() {
						defer wg.Done()
						ts.Clients[clientName].Start()
						ts.cancel()
					}()
				}
			} else {
				ts.Logger.Info("Client(%s) is exists.", clientName)
				return
			}
		} else {
			ts.Logger.Error("Client(%s) not exists.", clientName)
			return
		}
	}
	wg.Wait()
}
