package main

import (
	"context"
	"fmt"
	"sync"
)

func main() {
	logger := NewLogger(1, "main")

	config, err := LoadConfig("/etc/mysqlsync/config.yml")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// hjdb := NewHJDB(config.HJDB.LogLevel, config.HJDB.Addr, config.Name)

	// metricDirector = NewMetricDirector(config.Replication.LogLevel, hjdb)

	metricCh := make(chan interface{})
	moCh := make(chan MysqlOperation, 1000)
	gsCh := make(chan string)

	go func() {
		for range metricCh {

		}
	}()

	replication, err := NewReplication(config.Name, config.Replication, metricCh, moCh, gsCh, config.HJDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to NewReplication: %v", err))
	}

	replication.start(ctx, cancel)
}

func NewReplication(
	name string,
	rc ReplicationConfig,
	metricCh chan<- interface{},
	moCh chan MysqlOperation,
	gtidsetCh chan string,
	hc HJDBConfig,
) (*Replication, error) {
	return &Replication{
		tcpServer:     NewTCPServer(rc.LogLevel, rc.TCPAddr, moCh, gtidsetCh, metricCh),
		binlogExtract: NewBinlogExtract(rc.LogLevel, rc, moCh, gtidsetCh, metricCh),
		logger:        NewLogger(rc.LogLevel, "replication"),
		gtidsetCh:     gtidsetCh,
		moCh:          moCh,
	}, nil
}

type Replication struct {
	tcpServer     *TCPServer
	binlogExtract *BinlogExtract
	metric        *MetricDirector

	gtidsetCh chan string
	moCh      chan MysqlOperation

	cancel context.CancelFunc

	logger *Logger
}

func (repl *Replication) start(ctx context.Context, cancel context.CancelFunc) error {
	metricCh := make(chan MetricUnit)

	var childCtx context.Context
	var childCancel context.CancelFunc = func() {}
	var wg sync.WaitGroup

	go func() {
		if err := repl.tcpServer.Start(ctx); err != nil {
			repl.logger.Error("tcp server start failed: " + err.Error())
			cancel()
		}
	}()

	for {
		select {
		case gtidset := <-repl.gtidsetCh:
			childCancel()
			wg.Wait()

			// go func() {
			// 	for {
			// 		select {
			// 		case <-repl.moCh:
			// 			fmt.Println("xxxxxxx")
			// 		default:
			// 			return
			// 		}
			// 	}
			// }()

			childCtx, childCancel = context.WithCancel(context.Background())

			wg.Add(1)
			go func(gtid string) {
				defer wg.Done()
				if err := repl.binlogExtract.start(childCtx, gtid); err != nil { // 假设 RestartSync 需要 GTID 作为参数
					repl.logger.Error("failed to restart sync: " + err.Error())
				}

				repl.logger.Info("binlog extract stopped.")
			}(gtidset)
		case <-ctx.Done():
			childCancel()
			wg.Wait()
			return ctx.Err()
		}
	}
}
