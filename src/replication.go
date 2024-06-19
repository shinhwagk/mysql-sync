package main

import (
	"context"
	"sync"
	"time"
)

func NewReplication(name string, rc ReplicationConfig) *Replication {
	return &Replication{
		name:   name,
		rc:     rc,
		Logger: NewLogger(rc.LogLevel, "replication"),
	}
}

type Replication struct {
	name   string
	rc     ReplicationConfig
	Logger *Logger
}

func (repl *Replication) start(ctx context.Context, cancel context.CancelFunc) error {
	metricCh := make(chan MetricUnit)
	moCh := make(chan MysqlOperation)
	// gtidsets
	gsCh := make(chan string)
	defer close(metricCh)
	defer close(moCh)
	defer close(gsCh)

	metricDirector := NewMetricDirector(repl.rc.LogLevel, "replication", metricCh)
	tcpServer := NewTCPServer(repl.rc.LogLevel, repl.rc.TCPAddr, gsCh, moCh, metricCh)
	binlogExtract := NewBinlogExtract(repl.rc.LogLevel, repl.rc, gsCh, moCh, metricCh)

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

	var wg sync.WaitGroup
	childCtx, childCancel := context.WithCancel(context.Background())

	cleanSignal := true

	for {
		select {
		case gtidsets := <-gsCh:
			repl.Logger.Info("Got gtidsets: " + gtidsets)
			childCancel()
			wg.Wait()

			// new start
			childCtx, childCancel = context.WithCancel(context.Background())

			wg.Add(1)
			go func() {
				defer wg.Done()
				<-childCtx.Done()
				repl.Logger.Info("Clean mysql operation channel.")
				for cleanSignal {
					select {
					case <-moCh:
						repl.Logger.Debug("Consume mysql operation.")
					case <-time.After(time.Second):
					}
				}
				cleanSignal = true
			}()

			wg.Add(1)
			go func(gss string) {
				defer wg.Done()
				binlogExtract.Logger.Info("binlog extract Started.")
				if err := binlogExtract.Start(childCtx, gss); err != nil { // 假设 RestartSync 需要 GTID 作为参数
					repl.Logger.Error("failed to restart sync: " + err.Error())
				}
				cleanSignal = false
				binlogExtract.Logger.Info("binlog extract stopped.")
			}(gtidsets)
		case <-ctx.Done():
			childCancel()
			wg.Wait()
			return ctx.Err()
		}
	}
}
