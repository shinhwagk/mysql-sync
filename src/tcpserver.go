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

	moCh     <-chan MysqlOperation
	metricCh chan<- MetricUnit

	Clients map[string]*TCPServerClient

	BatchID uint
}

func NewTCPServer(logLevel int, listenAddress string, clientNames []string, moCh <-chan MysqlOperation, metricCh chan<- MetricUnit) *TCPServer {
	clients := make(map[string]*TCPServerClient)

	for _, name := range clientNames {
		clients[name] = NewTcpServerClient1(logLevel, name, metricCh)
	}

	fmt.Println("clients", len(clients))

	return &TCPServer{
		Logger:        NewLogger(logLevel, "tcp server"),
		listenAddress: listenAddress,

		moCh:     moCh,
		metricCh: metricCh,

		Clients: clients,
		BatchID: 0,
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
		listener.Close()
		for _, client := range ts.Clients {
			if client != nil {
				client.SetClose()
			}
		}
	}()

	go func() {
		if err := ts.distributor(); err != nil {
			fmt.Println(err.Error())
			listener.Close()
		}
	}()

	for {
		conn, err := listener.Accept()
		ts.Logger.Info("Listener started.")

		if err != nil {
			ts.Logger.Info("Error accepting:" + err.Error())
			return err
		}

		go ts.handleClient(conn)
	}
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
				client.pushClient()
			}
			return false
		}

		if client.State != ClientFree {
			return false
		}
	}
	return true
}

func (ts *TCPServer) distributor() error {
	maxRcCnt := cap(ts.moCh)
	const minRcCnt int = 10

	lastSecondCount := 0

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	ticker2 := time.NewTicker(time.Minute * 1)
	defer ticker2.Stop()

	tickerTimestamp := time.Now()

	fetchDelay := 0
	noReadyMs := 0

	lastDelay := 0
	fetchCount := 10
	for {
		if ts.clientsReady() {
			ts.Logger.Debug("MoCh cache capacity: %d/%d.", len(ts.moCh), maxRcCnt)
			delay := int(time.Since(tickerTimestamp).Milliseconds())

			if delay <= lastDelay {
				if fetchDelay <= lastDelay*20/100+lastDelay {
					fetchCount += minRcCnt
				} else {
					fetchCount = ((fetchCount / ((lastDelay*20/100 + lastDelay) / fetchDelay)) / 10) * 10
				}
			} else {
				fetchCount -= minRcCnt
			}

			fetchCount = max(minRcCnt, fetchCount)
			fmt.Println("xxxxxxx", fetchCount, lastDelay, delay, fetchDelay)

			lastDelay = delay

			// select {
			// // case <-ts.ctx.Done():
			// // 	ts.Logger.Info("Context cancelled, stopping handleFromServer loop.")
			// // 	return
			// case <-ticker.C:
			// 	remainingCapacity := maxRcCnt - len(ts.moCh)
			// 	ts.Logger.Debug("MoCh remaining capacity: %d/%d.", remainingCapacity, maxRcCnt)
			// 	hundredSendCount := max(minRcCnt, (lastSecondCount/minRcCnt)*minRcCnt/initDelay)
			// 	if hundredSendCount >= 10000 {
			// 		fetchCount = 1000
			// 	} else if hundredSendCount >= 1000 || hundredSendCount >= 100 {
			// 		fetchCount = 100
			// 	} else {
			// 		fetchCount = 10
			// 	}
			// 	ts.Logger.Debug("lastSecondCount %d hundredSendCount %d elapsed %d", lastSecondCount, hundredSendCount, initDelay)

			// 	lastSecondCount = 0
			// // case <-ticker2.C:

			// default:
			// 	fetchCount = 10
			// }
			fetchTimestamp := time.Now()
			if mos, err := ts.fetchMos(fetchCount); err != nil {
				ts.Logger.Error("Fetch mos fetch count: %d err: %s", fetchCount, err.Error())
				return err
			} else {
				fetchDelay = int(time.Since(fetchTimestamp).Milliseconds())
				fmt.Println("fetchcccccc", int(time.Since(fetchTimestamp).Milliseconds()))
				ts.BatchID += 1
				ts.Logger.Info("Push batch(%d) mos(%d) to clients.", ts.BatchID, len(mos))
				tickerTimestamp = time.Now()
				ts.pushClients(mos)
				lastSecondCount += len(mos)
			}
			noReadyMs = 0
		} else {
			time.Sleep(time.Millisecond * time.Duration(noReadyMs))
			ts.Logger.Debug("Client not ready sleep: %d.", noReadyMs)
			if noReadyMs <= 1000 {
				noReadyMs += 10
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
			cli.setPush(ts.BatchID, mos)
			ts.Logger.Info("Push to Client(%s) Batch(%d).", name, ts.BatchID)
			cli.pushClient()
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

func (ts *TCPServer) handleClient(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			fmt.Printf("Failed to read from client: %v", err)
		} else {
			fmt.Println("Client disconnected before sending a name")
		}
		return
	}

	clientName := scanner.Text()

	if client, exists := ts.Clients[clientName]; exists {
		if client.Dead {
			if err := client.InitConn(conn); err != nil {
				ts.Logger.Error("init client err: %s", err.Error())
			} else {
				ts.Logger.Info("Client %s Running.", clientName)
				client.Running()
			}
		} else {
			ts.Logger.Info("Client(%s) is Running.", clientName)
		}
	} else {
		ts.Logger.Error("client not exists: %s", clientName)
	}
}
