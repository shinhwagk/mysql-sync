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
		clients[name] = nil
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
	for _, client := range ts.Clients {
		if client == nil {
			return false
		}
		if client.State == ClientBusy {
			return false
		}
	}
	return true
}

func (ts *TCPServer) distributor() error {
	maxRcCnt := cap(ts.moCh)
	const minRcCnt int = 100

	lastSecondCount := 0

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	noReadyMs := 100

	for {
		if ts.clientsReady() && len(ts.moCh) >= 10 {
			fmt.Println("11115")
			sendCount := 10

			select {
			// case <-ts.ctx.Done():
			// 	ts.Logger.Info("Context cancelled, stopping handleFromServer loop.")
			// 	return
			case <-ticker.C:
				remainingCapacity := maxRcCnt - len(ts.moCh)
				ts.Logger.Debug("MoCh remaining capacity: %d/%d.", remainingCapacity, maxRcCnt)
				hundredSendCount := max(minRcCnt, (lastSecondCount/minRcCnt)*minRcCnt)
				if hundredSendCount >= 10000 {
					sendCount = 1000
				} else if hundredSendCount >= 1000 {
					sendCount = 100
				} else {
					sendCount = 10
				}
			default:
				sendCount = 10
			}
			fmt.Println("pushpushpushpushpushpushpush", sendCount)

			if mos, err := ts.fetchMos(sendCount); err != nil {
				fmt.Println(11112, err.Error())
				return err
			} else {
				fmt.Println("push", len(mos))
				if err := ts.pushClients(mos); err != nil {
					fmt.Println(11112, err.Error())
					return err
				} else {
					lastSecondCount = 0
				}
			}

			noReadyMs = 100
		} else {
			time.Sleep(time.Millisecond * time.Duration(noReadyMs))
			noReadyMs += 10
		}
	}
}

func (ts *TCPServer) pushClients(mos []MysqlOperation) error {
	ts.BatchID += 1

	var wg sync.WaitGroup

	for _, client := range ts.Clients {
		wg.Add(1)
		fmt.Println("clientxxxxxxx", client.Name)
		go func(cli *TCPServerClient) {
			defer wg.Done()
			cli.pushClient(ts.BatchID, mos)
		}(client)
	}
	wg.Wait()
	return nil
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

	_, ok := ts.Clients[clientName]
	if ok {
		fmt.Println("nononononononnono2", clientName)
	} else {
		fmt.Println("nononononononnono1", clientName)
	}

	if client, err := NewTcpServerClient(ts.Logger.Level, clientName, conn); err != nil {
		fmt.Println("xxxxxx" + err.Error())
	} else {
		ts.Clients[clientName] = client
		client.Running()
	}
}
