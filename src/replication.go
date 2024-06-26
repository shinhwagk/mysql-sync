package main

import (
	"context"
	"fmt"
)

func NewReplication(msc MysqlSyncConfig) *Replication {
	return &Replication{
		msc:    msc,
		Logger: NewLogger(msc.Replication.LogLevel, "replication"),
	}
}

type Replication struct {
	msc    MysqlSyncConfig
	Logger *Logger
}

func (repl *Replication) start(ctx context.Context, cancel context.CancelFunc) error {
	metricCh := make(chan MetricUnit)

	cacheSize := 1000
	if repl.msc.Replication.Settings != nil && repl.msc.Replication.Settings.CacheSize > cacheSize {
		cacheSize = repl.msc.Replication.Settings.CacheSize
	}
	moCh := make(chan MysqlOperation, cacheSize)

	defer close(metricCh)
	defer close(moCh)

	destNames := []string{}

	if len(repl.msc.Destination.Destinations) == 0 {
		return fmt.Errorf("dest number 0")
	}

	destGtidSetss := make([]map[string]uint, len(repl.msc.Destination.Destinations))
	for destName, dc := range repl.msc.Destination.Destinations {
		gss := NewGtidSets(repl.msc.HJDB.Addr, repl.msc.Replication.Name, destName)
		gss.InitStartupGtidSetsMap(dc.InitGtidSetsRangeStr)
		destGtidSetss = append(destGtidSetss, gss.GtidSetsMap)

		fmt.Println("xxxxxxx", destName)
		destNames = append(destNames, destName)
	}

	metricDirector := NewMetricDirector(repl.msc.Replication.LogLevel, "replication", metricCh)
	tcpServer := NewTCPServer(repl.msc.Replication.LogLevel, repl.msc.Replication.TCPAddr, destNames, moCh, metricCh)
	binlogExtract := NewBinlogExtract(repl.msc.Replication.LogLevel, repl.msc.Replication, moCh, metricCh)

	initGtidSetsRangeStr := GetGtidSetsRangeStrFromGtidSetsMap(MergeGtidSets(destGtidSetss))

	repl.Logger.Info("Init gtidsets range string:'%s'", initGtidSetsRangeStr)

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

	binlogExtract.Logger.Info("binlog extract Started.")
	if err := binlogExtract.Start(ctx, initGtidSetsRangeStr); err != nil { // 假设 RestartSync 需要 GTID 作为参数
		binlogExtract.Logger.Error("failed to restart sync: " + err.Error())
	}
	binlogExtract.Logger.Info("binlog extract stopped.")
	return nil
}
