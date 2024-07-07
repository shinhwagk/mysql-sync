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
	defer close(metricCh)

	cacheSize := 1000
	if repl.msc.Replication.Settings != nil && repl.msc.Replication.Settings.CacheSize > cacheSize {
		cacheSize = repl.msc.Replication.Settings.CacheSize
	}
	repl.Logger.Info("Settings cache size: %d", cacheSize)
	moCh := make(chan MysqlOperation, cacheSize)
	defer close(moCh)

	destNames := []string{}

	if len(repl.msc.Destination.Destinations) == 0 {
		repl.Logger.Error("dest number 0")
		return
	}

	destGtidSetss := make([]map[string]uint, len(repl.msc.Destination.Destinations))
	for destName, dc := range repl.msc.Destination.Destinations {
		gss := NewGtidSets(repl.msc.HJDB.Addr, repl.msc.Replication.Name, destName)
		if err := gss.InitStartupGtidSetsMap(dc.Sync.InitGtidSetsRangeStr); err != nil {
			return
		}
		destGtidSetss = append(destGtidSetss, gss.GtidSetsMap)
		destNames = append(destNames, destName)
	}
	initGtidSetsRangeStr := GetGtidSetsRangeStrFromGtidSetsMap(MergeGtidSetss(destGtidSetss))
	repl.Logger.Info("Init gtidsets range string:'%s'", initGtidSetsRangeStr)

	metricDirector := NewMetricReplDirector(repl.msc.Replication.LogLevel, "replication", repl.msc.Replication.Name, metricCh)
	tcpServer := NewTCPServer(repl.msc.Replication.LogLevel, repl.msc.Replication.TCPAddr, destNames, moCh, metricCh)
	extract := NewBinlogExtract(repl.msc.Replication.LogLevel, repl.msc.Replication, moCh, metricCh)

	metricCtx, metricCancel := context.WithCancel(context.Background())
	defer metricCancel()
	var wg0 sync.WaitGroup
	wg0.Add(1)
	go func() {
		defer wg0.Done()
		promExportPort := 9092
		if repl.msc.Replication.Prometheus != nil {
			if repl.msc.Replication.Prometheus.ExportPort != 0 {
				promExportPort = repl.msc.Replication.Prometheus.ExportPort
			} else {
				repl.Logger.Error("prometheus export port %d.", repl.msc.Replication.Prometheus.ExportPort)
				return
			}
		}
		metricDirector.Start(metricCtx, fmt.Sprintf("0.0.0.0:%d", promExportPort))
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
		extract.Start(ctx, initGtidSetsRangeStr)
		cancel()
	}()
	wg.Wait()
	fmt.Println("sssss")

	metricCancel()
	// close(metricCh)
	wg0.Wait()
}
