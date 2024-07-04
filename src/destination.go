package main

import (
	"context"
	"fmt"
	"sync"
)

func NewDestination(replName string, destName string, tcpAddr string, dc DestinationConfig, hjdbAddr string) *Destination {
	return &Destination{
		replName: replName,
		destName: destName,
		tcpAddr:  tcpAddr,
		dc:       dc,
		hjdbAddr: hjdbAddr,
		Logger:   NewLogger(dc.LogLevel, "destination"),
	}
}

type Destination struct {
	replName string
	destName string
	tcpAddr  string
	dc       DestinationConfig
	hjdbAddr string
	Logger   *Logger
}

func (dest *Destination) Start(ctx context.Context, cancel context.CancelFunc) error {
	metricCh := make(chan MetricUnit)
	cacheSize := 1000
	if dest.dc.Sync.CacheSize > cacheSize {
		cacheSize = dest.dc.Sync.CacheSize
	}
	moCh := make(chan MysqlOperation, dest.dc.Sync.CacheSize)
	defer close(metricCh)
	defer close(moCh)

	// gtidsets must first init.
	gtidSets := NewGtidSets(dest.hjdbAddr, dest.replName, dest.destName)
	err := gtidSets.InitStartupGtidSetsMap(dest.dc.Sync.InitGtidSetsRangeStr)
	if err != nil {
		return err
	}

	metricDirector := NewMetricDestDirector(dest.dc.LogLevel, "destination", dest.replName, dest.destName, metricCh)
	tcpClient, err := NewTCPClient(dest.dc.LogLevel, dest.tcpAddr, dest.destName, moCh, metricCh)
	if err != nil {
		tcpClient.Logger.Error("NewTCPClient: %s", err.Error())
		return err
	}

	mysqlClient, err := NewMysqlClient(dest.dc.LogLevel, dest.dc.Mysql.Dsn, dest.dc.Mysql.SkipErrors)
	if err != nil {
		mysqlClient.Logger.Error("NewMysqlClient: ", err.Error())
		return err
	}
	defer mysqlClient.Close()

	replicateFilter := NewReplicateFilter(dest.dc.Sync.Replicate)
	mysqlApplier := NewMysqlApplier(dest.dc.LogLevel, gtidSets, mysqlClient, replicateFilter, metricCh)

	mdCtx, mdCancel := context.WithCancel(context.Background())
	defer mdCancel()

	var wg0 sync.WaitGroup
	wg0.Add(1)
	go func() {
		defer wg0.Done()
		metricDirector.Logger.Info("started.")
		promExportPort := 9092
		if dest.dc.Prometheus != nil {
			promExportPort = dest.dc.Prometheus.ExportPort
		}
		metricDirector.Start(mdCtx, fmt.Sprintf("0.0.0.0:%d", promExportPort))
		metricDirector.Logger.Info("stopped.")
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		mysqlApplier.Logger.Info("started.")
		mysqlApplier.Start(ctx, moCh)
		mysqlApplier.Logger.Info("stopped.")
		cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tcpClient.Logger.Info("started.")
		tcpClient.Start(ctx)
		tcpClient.Logger.Info("stopped.")
		cancel()
	}()
	wg.Wait()

	mdCancel()
	wg0.Wait()

	dest.Logger.Info("stopped。")

	return nil
}
