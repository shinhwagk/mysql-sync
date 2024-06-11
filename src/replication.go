package main

import (
	"context"
	"fmt"
	"sync"
)

func main() {
	logger := NewLogger(1, "main")

	config, err := LoadConfig("/workspaces/mysqlbinlog-sync/src/config.yml")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// hjdb := NewHJDB(config.HJDB.LogLevel, config.HJDB.Addr, config.Name)

	// metricDirector = NewMetricDirector(config.Replication.LogLevel, hjdb)

	metricCh := make(chan interface{}, 100)

	go func() {
		for range metricCh {

		}
	}()

	replication, err := NewReplication(config.Name, config.Replication, metricCh, config.HJDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to NewReplication: %v", err))
	}

	replication.start(ctx, cancel)
}

func NewReplication(name string, rc ReplicationConfig, metricCh chan<- interface{}, hc HJDBConfig) (*Replication, error) {
	return &Replication{
		tcpServer:     NewTCPServer(rc.LogLevel, rc.TCPAddr, metricCh),
		binlogExtract: NewBinlogExtract(rc.LogLevel, rc, metricCh),
		logger:        NewLogger(rc.LogLevel, "replication"),
	}, nil
}

type Replication struct {
	tcpServer     *TCPServer
	binlogExtract *BinlogExtract
	metric        *MetricDirector

	cancel context.CancelFunc

	logger *Logger
}

func (repl *Replication) start(ctx context.Context, cancel context.CancelFunc) error {
	moCh := make(chan MysqlOperation, 100)
	gsCh := make(chan string)

	var childCtx context.Context
	var childCancel context.CancelFunc = func() {}
	var wg sync.WaitGroup

	go func() {
		if err := repl.tcpServer.Start(ctx, moCh, gsCh); err != nil {
			repl.logger.Error("tcp server start failed: " + err.Error())
			cancel()
		}
	}()

	for {
		select {
		case gtidset := <-gsCh:
			childCancel()
			wg.Wait()

			go func() {
				for {
					select {
					case <-moCh:
					default:
						return
					}
				}
			}()

			childCtx, childCancel = context.WithCancel(context.Background())

			wg.Add(1)
			go func(gtid string) {
				defer wg.Done()
				if err := repl.binlogExtract.start(childCtx, gtid, moCh); err != nil { // 假设 RestartSync 需要 GTID 作为参数
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
