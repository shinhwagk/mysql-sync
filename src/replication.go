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

	replication, err := NewReplication(config.Name, config.Replication)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to NewReplication: %v", err))
	}

	replication.start(ctx, cancel)
}

func NewReplication(name string, rc ReplicationConfig) (*Replication, error) {
	return &Replication{
		name:   name,
		rc:     rc,
		Logger: NewLogger(rc.LogLevel, "replication"),
	}, nil
}

type Replication struct {
	name   string
	rc     ReplicationConfig
	Logger *Logger
}

func (repl *Replication) start(ctx context.Context, cancel context.CancelFunc) error {
	metricCh := make(chan MetricUnit)
	moCh := make(chan MysqlOperation, 1000)
	// gtidset
	gsCh := make(chan string)
	defer close(metricCh)
	defer close(moCh)
	defer close(gsCh)

	metricDirector := NewMetricDirector(repl.rc.LogLevel, "replication", metricCh)
	tcpServer := NewTCPServer(repl.rc.LogLevel, repl.rc.TCPAddr, gsCh, moCh, metricCh)
	binlogExtract := NewBinlogExtract(repl.rc.LogLevel, repl.rc, gsCh, moCh, metricCh)

	var childCtx context.Context
	var childCancel context.CancelFunc = func() {}
	var wg sync.WaitGroup

	go func() {
		if err := tcpServer.Start(ctx); err != nil {
			repl.Logger.Error("tcp server start failed: " + err.Error())
			cancel()
		}
	}()

	go func() {
		metricDirector.Logger.Info("started.")
		metricDirector.Start(ctx, "0.0.0.0:9091")
		metricDirector.Logger.Info("stopped.")
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
						fmt.Println("xxxxxxx")
					default:
						return
					}
				}
			}()

			childCtx, childCancel = context.WithCancel(context.Background())

			wg.Add(1)
			go func(gtid string) {
				defer wg.Done()
				if err := binlogExtract.start(childCtx, gtidset); err != nil { // 假设 RestartSync 需要 GTID 作为参数
					repl.Logger.Error("failed to restart sync: " + err.Error())
				}

				repl.Logger.Info("binlog extract stopped.")
			}(gtidset)
		case <-ctx.Done():
			childCancel()
			wg.Wait()
			return ctx.Err()
		}
	}
}
