package main

import (
	"bufio"
	"context"
	"encoding/gob"
	"net"
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
	gob.Register(MysqlOperationBinLogPos{})

	gob.Register(Signal1{})
	gob.Register(Signal2{})
}

type TCPServer struct {
	Logger        *Logger
	listenAddress string
	ctx           context.Context
	cancel        context.CancelFunc
	moCh          <-chan MysqlOperation
	metricCh      chan<- MetricUnit
	Clients       map[string]*TCPServerClient
	BatchID       uint
	dsgsrsCh      chan<- DestStartGtidSetsRangeStr
}

func NewTCPServer(logLevel int, listenAddress string, clientNames []string, moCh <-chan MysqlOperation, metricCh chan<- MetricUnit, destStartGtidSetsStrCh chan<- DestStartGtidSetsRangeStr) *TCPServer {
	clients := make(map[string]*TCPServerClient)

	for _, name := range clientNames {
		clients[name] = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &TCPServer{
		Logger:        NewLogger(logLevel, "tcp server"),
		listenAddress: listenAddress,
		ctx:           ctx,
		cancel:        cancel,
		moCh:          moCh,
		metricCh:      metricCh,
		Clients:       clients,
		BatchID:       0,
		dsgsrsCh:      destStartGtidSetsStrCh,
	}
}

func (ts *TCPServer) Start(ctx context.Context) {
	ts.Logger.Info("Started.")
	defer ts.Logger.Info("Closed.")

	listener, err := net.Listen("tcp", ts.listenAddress)
	if err != nil {
		ts.Logger.Error("Listening: %s.", err)

	}

	go func() {
		<-ctx.Done()
		ts.cancel()
	}()

	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ts.ctx.Done()
		// clear clients
		for {
			allDead := true
			for _, client := range ts.Clients {
				if client != nil {
					client.cancel()
					allDead = allDead && client.Dead
				}
			}
			if allDead {
				break
			}
		}
		ts.Logger.Info("Clients all cleanup.")

		// close listener
		if err := listener.Close(); err != nil {
			ts.Logger.Error("Listener Close: %s.", err)
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
}

func (ts *TCPServer) clientsReady() bool {
	for name, client := range ts.Clients {
		if client == nil || client.Dead {
			return false
		}

		if client.State == ClientBusy {
			if client.SendError != nil {
				ts.Logger.Info("Push to Client(%s) Batch(%d) Mos(%d) again.", name, client.SendBatchID, len(client.SendMos))
				client.ClientPush()
			}
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

	sendStartTs := time.Now()

	noReadyMs := 0

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
				if noReadyMs < 1000 {
					noReadyMs += 10
				}
				ts.Logger.Debug("Clients not ready sleep: %dms.", noReadyMs)
				time.Sleep(time.Millisecond * time.Duration(noReadyMs))
				continue
			}
			noReadyMs = 0

			ts.Logger.Debug("MoCh cache capacity: %d/%d.", len(ts.moCh), maxRcCnt)
			sendDelayMs := max(int(time.Since(sendStartTs).Milliseconds()), 1) // min delay 1ms

			if fetchCount >= minRcCnt {
				if resetBaseLine {
					sendBaseLineDelayMs = int(float64(sendDelayMs) * 1.2)
					resetBaseLine = false
					fetchCount = max(sendBaseLineMaxCount, fetchCount)
					ts.Logger.Debug("Adaptive sending params(reset base line): sendDelayMs(%d), sendBaseLineDelayMs(%d), sendBaseLineMaxCount(%d), fetchCount(%d).", sendDelayMs, sendBaseLineDelayMs, sendBaseLineMaxCount, fetchCount)

				} else {
					delayMsSlice = updateSlice(delayMsSlice, sendDelayMs)

					select {
					case <-ticker.C:
						fetchCount = minRcCnt
						resetBaseLine = true
					default:
						avgSendDelayMs := calculateAdjustedMean(delayMsSlice)
						if avgSendDelayMs <= sendBaseLineDelayMs {
							sendBaseLineMaxCount = max(fetchCount, sendBaseLineMaxCount)
							fetchCount += minRcCnt
						} else {
							sendBaseLineMaxCount = max(sendBaseLineMaxCount-minRcCnt, minRcCnt) // not less than minRcCnt
							fetchCount = max(fetchCount-minRcCnt, minRcCnt)                     // not less than minRcCnt
						}
						fetchCount = max(fetchCount, sendBaseLineMaxCount)
						ts.Logger.Debug("Adaptive sending params: sendDelayMs(%d), avgSendDelayMs(%d), sendBaseLineDelayMs(%d), sendBaseLineMaxCount(%d), fetchCount(%d).", sendDelayMs, avgSendDelayMs, sendBaseLineDelayMs, sendBaseLineMaxCount, fetchCount)
					}
				}
			}

			fetchCount = min(fetchCount, len(ts.moCh))
			// multiples of minRcCnt
			fetchCount = fetchCount / minRcCnt * minRcCnt
			// Ensure fetchCount is never less than minRcCnt. This line is critical and must remain here for correct functionality.
			fetchCount = max(fetchCount, minRcCnt)

			fetchStartTs := time.Now()
			mos := ts.fetchMos(fetchCount)
			fetchElapsedMs := int(time.Since(fetchStartTs).Milliseconds())
			ts.Logger.Debug("Fetch mos(%d) elapsed ms(%d)", fetchCount, fetchElapsedMs)
			ts.BatchID += 1
			sendStartTs = time.Now()
			ts.ClientsPush(mos)
		}
	}
}

func (ts *TCPServer) ClientsPush(mos []MysqlOperation) {
	ts.Logger.Info("Push to Clients Batch(%d) Mos(%d).", ts.BatchID, len(mos))
	var wg sync.WaitGroup
	for name, client := range ts.Clients {
		wg.Add(1)
		go func(cli *TCPServerClient) {
			defer wg.Done()
			cli.SetPush(ts.BatchID, mos)
			ts.Logger.Info("Push to Client(%s) Batch(%d) Mos(%d).", name, ts.BatchID, len(mos))
			cli.ClientPush()
		}(client)
	}
	wg.Wait()
}

func (ts *TCPServer) fetchMos(count int) []MysqlOperation {
	mos := make([]MysqlOperation, 0, count)
	for i := 0; i < count; i++ {
		mos = append(mos, <-ts.moCh)
	}
	return mos
}

func (ts *TCPServer) handleClients(listener net.Listener) {
	ts.Logger.Info("HandleClients started.")
	defer ts.Logger.Info("HandleClients closed.")

	var wg sync.WaitGroup
	defer wg.Wait()

	for i := 0; i < len(ts.Clients); i++ {
		conn, err := listener.Accept()
		if err != nil {
			ts.Logger.Error("Accepting: %s.", err)
			break
		}

		scanner := bufio.NewScanner(conn)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				ts.Logger.Error("Failed to read client name: %s.", err)
				break
			}
		}

		initClientInfo := strings.Split(scanner.Text(), "@")
		clientName := initClientInfo[0]
		ts.dsgsrsCh <- DestStartGtidSetsRangeStr{DestName: clientName, GtidSetsStr: initClientInfo[1]}

		if client, exists := ts.Clients[clientName]; exists {
			if client == nil {
				if ts.Clients[clientName], err = NewTcpServerClient(ts.Logger.Level, clientName, ts.metricCh, conn); err != nil {
					ts.Logger.Error("Client(%s) Init: %s.", clientName, err)
				} else {
					wg.Add(1)
					go func() {
						defer wg.Done()
						ts.Clients[clientName].Start()
						ts.cancel()
					}()
					continue
				}
			} else {
				ts.Logger.Error("Client(%s) was exists.", clientName)
			}
		} else {
			ts.Logger.Error("Client(%s) is not exists.", clientName)
		}
		break
	}
	ts.Logger.Info("All clients are ready.")
}
