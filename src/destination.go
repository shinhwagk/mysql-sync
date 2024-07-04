package main

import (
	"context"
	"fmt"
	"sync"
)

func NewDestination(msc MysqlSyncConfig, destName string) *Destination {
	destConf := msc.Destination.Destinations[destName]
	return &Destination{
		msc:    msc,
		Name:   destName,
		Logger: NewLogger(destConf.LogLevel, "destination"),
	}
}

type Destination struct {
	msc    MysqlSyncConfig
	Name   string
	Logger *Logger
}

func (dest *Destination) Start(ctx context.Context, cancel context.CancelFunc) error {
	replName := dest.msc.Replication.Name
	destConf := dest.msc.Destination.Destinations[dest.Name]
	hjdbAddr := dest.msc.HJDB.Addr
	tcpAddr := dest.msc.Destination.TCPAddr

	metricCh := make(chan MetricUnit)
	cacheSize := 1000
	if dest.msc.Destination.CacheSize > cacheSize {
		cacheSize = dest.msc.Destination.CacheSize
	}
	moCh := make(chan MysqlOperation, cacheSize)
	defer close(metricCh)
	defer close(moCh)

	// gtidsets must first init.
	gtidSets := NewGtidSets(hjdbAddr, replName, dest.Name)
	err := gtidSets.InitStartupGtidSetsMap(destConf.Sync.InitGtidSetsRangeStr)
	if err != nil {
		return err
	}

	metricDirector := NewMetricDestDirector(destConf.LogLevel, "destination", replName, dest.Name, metricCh)
	tcpClient, err := NewTCPClient(destConf.LogLevel, tcpAddr, dest.Name, moCh, metricCh)
	if err != nil {
		tcpClient.Logger.Error("NewTCPClient: %s", err.Error())
		return err
	}

	mysqlClient, err := NewMysqlClient(destConf.LogLevel, destConf.Mysql.Dsn, destConf.Mysql.SkipErrors)
	if err != nil {
		mysqlClient.Logger.Error("NewMysqlClient: ", err.Error())
		return err
	}
	defer mysqlClient.Close()

	replicateFilter := NewReplicateFilter(destConf.Sync.Replicate)
	mysqlApplier := NewMysqlApplier(destConf.LogLevel, gtidSets, mysqlClient, replicateFilter, metricCh)

	mdCtx, mdCancel := context.WithCancel(context.Background())
	defer mdCancel()

	var wg0 sync.WaitGroup
	wg0.Add(1)
	go func() {
		defer wg0.Done()
		metricDirector.Logger.Info("started.")
		promExportPort := 9092
		if destConf.Prometheus != nil {
			promExportPort = destConf.Prometheus.ExportPort
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
