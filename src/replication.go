package main

import (
	"context"
	"fmt"
	"sync"
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

func (repl *Replication) start(ctx context.Context, cancel context.CancelFunc) {
	repl.Logger.Info("Started.")
	defer repl.Logger.Info("Closed.")

	metricCh := make(chan MetricUnit)

	cacheSize := 1000
	if repl.msc.Replication.Settings != nil && repl.msc.Replication.Settings.CacheSize > cacheSize {
		cacheSize = repl.msc.Replication.Settings.CacheSize
	}
	repl.Logger.Info("Settings cache size: %d", cacheSize)
	moCh := make(chan MysqlOperation, cacheSize)

	defer close(metricCh)
	defer close(moCh)

	destNames := []string{}

	if len(repl.msc.Destination.Destinations) == 0 {
		repl.Logger.Error("dest number 0")
		return
	}

	destGtidSetss := make([]map[string]uint, len(repl.msc.Destination.Destinations))
	for destName, dc := range repl.msc.Destination.Destinations {
		gss := NewGtidSets(repl.msc.HJDB.Addr, repl.msc.Replication.Name, destName)
		gss.InitStartupGtidSetsMap(dc.Sync.InitGtidSetsRangeStr)
		destGtidSetss = append(destGtidSetss, gss.GtidSetsMap)

		fmt.Println("xxxxxxx", destName)
		destNames = append(destNames, destName)
	}

	metricDirector := NewMetricReplDirector(repl.msc.Replication.LogLevel, "replication", repl.msc.Replication.Name, metricCh)
	tcpServer := NewTCPServer(repl.msc.Replication.LogLevel, repl.msc.Replication.TCPAddr, destNames, moCh, metricCh)
	binlogExtract := NewBinlogExtract(repl.msc.Replication.LogLevel, repl.msc.Replication, moCh, metricCh)

	initGtidSetsRangeStr := GetGtidSetsRangeStrFromGtidSetsMap(MergeGtidSets(destGtidSetss))

	repl.Logger.Info("Init gtidsets range string:'%s'", initGtidSetsRangeStr)

	go func() {
		metricDirector.Start(ctx, "0.0.0.0:9091")
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tcpServer.Start(ctx)
		cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		binlogExtract.Start(ctx, initGtidSetsRangeStr)
		cancel()
	}()
	wg.Wait()
}
