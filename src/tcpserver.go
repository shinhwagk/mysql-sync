package main

import (
	"bufio"
	"context"
	"encoding/gob"
	"math"
	"net"
	"strings"
	"sync"
	"time"
)

func init() {
	gob.Register(MysqlOperationHeartbeat{})
	gob.Register(MysqlOperationGTID{})
	gob.Register(MysqlOperationDDLDatabase{})
	gob.Register(MysqlOperationDDLTable{})
	gob.Register(MysqlOperationDCLUser{})
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
				ts.Logger.Debug("Push to Client(%s) Batch(%d) Mos(%d) again.", name, client.SendBatchID, len(client.SendMos))
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

	maxCapacity := cap(ts.moCh)
	// const minRcCnt int = 10
	// maxRcCnt := (maxCapacity * 5) / 100 / minRcCnt * minRcCnt // 5%

	sendStartTs := time.Now()

	// const maxSendLatencyMs int = 10 * 1000

	noReadyMs := 0

	// sendThroughputBaseLine := float64(0)
	// sendBaseLineMaxCount := 0
	fetchCount := 0
	// fetchCountLast := 0

	// min delay 1ms
	// sendThroughputHistory := make([]float64, 10) // maxSize must greater then 3

	// moChFlood := false
	// moChFloodFactor := float64(1)

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	afc := NewAdaptiveFetchCount(ts.Logger, maxCapacity)

	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			if !ts.clientsReady() {
				noReadyMs = int(math.Min(float64(noReadyMs+10), 1000))
				ts.Logger.Debug("Clients not ready sleep: %d ms.", noReadyMs)
				time.Sleep(time.Millisecond * time.Duration(noReadyMs))
				continue
			}
			noReadyMs = 0

			ts.Logger.Debug("MoCh cache capacity: %d/%d.", len(ts.moCh), maxCapacity)

			sendLatencyMs := max(int(time.Since(sendStartTs).Milliseconds()), 1)

			fetchCount = afc.EvaluateFetchCount(sendLatencyMs, len(ts.moCh))

			for len(ts.moCh) < fetchCount {
				select {
				case <-ts.ctx.Done():
					return
				default:
					time.Sleep(time.Millisecond * 10)
					ts.Logger.Trace("Fetch mos(%d) waiting")
				}
			}

			fetchStartTs := time.Now()
			mos := ts.fetchMos(fetchCount)
			fetchElapsedMs := int(time.Since(fetchStartTs).Milliseconds())
			ts.Logger.Debug("Fetch mos(%d) from mo cache, elapsed ms(%d)", fetchCount, fetchElapsedMs)
			ts.BatchID += 1
			sendStartTs = time.Now()
			ts.ClientsPush(mos)
		}
	}
}

func (ts *TCPServer) ClientsPush(mos []MysqlOperation) {
	var wg sync.WaitGroup
	for name, client := range ts.Clients {
		wg.Add(1)
		go func(cli *TCPServerClient) {
			defer wg.Done()
			cli.SetPush(ts.BatchID, mos)
			ts.Logger.Debug("Push to Client(%s) Batch(%d) Mos(%d).", name, ts.BatchID, len(mos))
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

// Adaptive Fetch Count
const maxSendLatencyMs int = 10 * 1000 // 10s
const minRcCnt int = 10
const floodDampingFactor = 0.3
const increaseFactor = 0.1

// const decreaseFactor = 0.05

type AdaptiveFetchCount struct {
	Logger            *Logger
	throughputHistory []float64
	baseLineMaxCount  int
	flood             bool
	floodFactor       float64
	maxCapacity       int
	fetchCount        int
	damping           float64
}

func (afc *AdaptiveFetchCount) EvaluateFetchCount(sendLatencyMs int, filledCapacity int) int {
	_fetchCount := afc.fetchCount

	if _fetchCount == 0 {
		afc.fetchCount = minRcCnt
		return minRcCnt
	}

	if filledCapacity >= afc.maxCapacity*1/10 {
		afc.flood = true
		afc.floodFactor = float64(filledCapacity) / float64(afc.maxCapacity)
		afc.floodFactor = afc.floodFactor * floodDampingFactor
	} else {
		if afc.flood {
			_fetchCount = minRcCnt
			afc.flood = false
			afc.floodFactor = 0
		}
	}

	sendThroughput := float64(afc.fetchCount*1000) / float64(sendLatencyMs)
	afc.throughputHistory = updateSliceFloat64(afc.throughputHistory, sendThroughput)

	avgSendThroughput := calculateMeanWithoutMinMaxFloat64(afc.throughputHistory)
	// avgSendLatencyMs := calculateMeanWithoutMinMaxInt(afc.latencyMsHistory)

	afc.Logger.Debug("adaptive fetch -- sendThroughput %.4f", sendThroughput)
	afc.Logger.Debug("adaptive fetch -- avgSendThroughput %.4f", avgSendThroughput)
	afc.Logger.Debug("adaptive fetch -- sendLatencyMs %d", sendLatencyMs)
	// afc.Logger.Debug("adaptive fetch -- avgSendLatencyMs %d", avgSendLatencyMs)

	floodFactorThresholdThroughput := avgSendThroughput * (afc.floodFactor + 1)
	floodFactorThresholdLatencyMs := float64(maxSendLatencyMs) * (afc.floodFactor + 1)

	afc.Logger.Debug("adaptive fetch -- flood %t floodFactor %.4f", afc.flood, afc.floodFactor)
	afc.Logger.Debug("adaptive fetch -- filledCapacity %d", filledCapacity)

	if float64(sendLatencyMs) > floodFactorThresholdLatencyMs {
		afc.damping += float64(sendLatencyMs) / floodFactorThresholdLatencyMs * 0.1
		afc.Logger.Debug("adaptive fetch -- floodFactorThresholdLatencyMs %.4f", floodFactorThresholdLatencyMs)
	} else if sendThroughput > floodFactorThresholdThroughput {
		afc.damping += sendThroughput / floodFactorThresholdThroughput * 0.1
		afc.Logger.Debug("adaptive fetch -- floodFactorThresholdThroughput %.4f", floodFactorThresholdThroughput)
	}

	afc.damping = max(afc.damping, 0.2) - 0.2

	if afc.damping >= 0.1 {
		afc.Logger.Debug("adaptive fetch -- decrease %d", int(float64(afc.baseLineMaxCount)*afc.damping))
		afc.baseLineMaxCount -= max(int(float64(afc.baseLineMaxCount)*afc.damping), minRcCnt)
		_fetchCount = afc.baseLineMaxCount
		afc.damping = 0
	} else {
		_fetchCount += max(int(float64(_fetchCount)*increaseFactor), minRcCnt)
		afc.Logger.Debug("adaptive fetch -- increase _fetchCount %d", _fetchCount)

		afc.damping = max(afc.damping-0.1, 0)
	}

	afc.Logger.Debug("adaptive fetch -- damping %.4f", afc.damping)

	afc.Logger.Debug("adaptive fetch -- baseLineMaxCount %d", afc.baseLineMaxCount)

	afc.baseLineMaxCount = max(afc.baseLineMaxCount/minRcCnt*minRcCnt, minRcCnt)
	afc.baseLineMaxCount = max(afc.baseLineMaxCount, _fetchCount)

	_fetchCount = afc.baseLineMaxCount
	_fetchCount = min(_fetchCount, afc.maxCapacity/100) // up top 1%
	_fetchCount = min(_fetchCount, filledCapacity)
	// Adjust fetchCount to the largest multiple of minRcCnt that is not less than minRcCnt (this line is critical and must remain)
	_fetchCount = max(_fetchCount/minRcCnt*minRcCnt, minRcCnt)

	afc.fetchCount = _fetchCount

	afc.Logger.Debug("adaptive fetch -- fetchCount %d baseLineMaxCount %d ", _fetchCount, afc.baseLineMaxCount)

	return _fetchCount
}

func NewAdaptiveFetchCount(logger *Logger, maxCapacity int) *AdaptiveFetchCount {
	return &AdaptiveFetchCount{
		Logger:            logger,
		throughputHistory: make([]float64, 10),
		baseLineMaxCount:  0,
		flood:             false,
		floodFactor:       float64(0),
		maxCapacity:       maxCapacity,
		fetchCount:        0,
		damping:           0,
	}
}
