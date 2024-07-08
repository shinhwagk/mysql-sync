package main

import (
	"context"
	"fmt"
	"time"
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

func (dest *Destination) Start(ctx context.Context, cancel context.CancelFunc) {
	dest.Logger.Info("Started.")
	defer dest.Logger.Info("Closed.")

	replName := dest.msc.Replication.Name
	destConf := dest.msc.Destination.Destinations[dest.Name]
	hjdbAddr := dest.msc.HJDB.Addr
	tcpAddr := dest.msc.Destination.TCPAddr

	metricCh := make(chan MetricUnit)
	defer close(metricCh)
	cacheSize := 1000
	if dest.msc.Destination.CacheSize > cacheSize {
		cacheSize = dest.msc.Destination.CacheSize
	}
	moCh := make(chan MysqlOperation, cacheSize)
	defer close(moCh)

	// gtidsets must first init.
	gtidSets := NewGtidSets(hjdbAddr, replName, dest.Name)
	err := gtidSets.InitStartupGtidSetsMap(destConf.Sync.InitGtidSetsRangeStr)
	if err != nil {
		return
	}

	metricDirector := NewMetricDestDirector(destConf.LogLevel, "destination", replName, dest.Name, metricCh)
	tcpClient, err := NewTCPClient(destConf.LogLevel, tcpAddr, dest.Name, moCh, metricCh)
	if err != nil {
		tcpClient.Logger.Error("NewTCPClient: %s", err)
		return
	}

	mysqlClient, err := NewMysqlClient(destConf.LogLevel, destConf.Mysql)
	if err != nil {
		mysqlClient.Logger.Error("NewMysqlClient: %s", err)
		return
	}
	defer mysqlClient.Close()

	replicateFilter := NewReplicateFilter(destConf.Sync.Replicate)
	mysqlApplier := NewMysqlApplier(destConf.LogLevel, gtidSets, mysqlClient, replicateFilter, metricCh)

	ctxMa, cancelMa := context.WithCancel(context.Background())
	ctxTc, cancelTc := context.WithCancel(context.Background())
	ctxMd, cancelMd := context.WithCancel(context.Background())

	go func() {
		promExportPort := 9092
		if destConf.Prometheus != nil {
			if destConf.Prometheus.ExportPort != 0 {
				promExportPort = destConf.Prometheus.ExportPort
			} else {
				dest.Logger.Error("prometheus export port %d.", destConf.Prometheus.ExportPort)
				return
			}
		}
		metricDirector.Start(ctx, fmt.Sprintf("0.0.0.0:%d", promExportPort))
		cancel()
		cancelMd()
	}()

	go func() {
		mysqlApplier.Start(ctx, moCh)
		cancel()
		cancelMa()
	}()

	go func() {
		tcpClient.Start(ctx)
		cancel()
		cancelTc()
	}()

	<-ctxMd.Done()
Loop:
	for {
		select {
		case <-metricCh:
		case <-ctxMa.Done():
			break Loop
		case <-time.After(time.Millisecond * 10):
		}
	}

	for {
		select {
		case <-moCh:
		case <-metricCh:
		case <-ctxTc.Done():
			return
		case <-time.After(time.Millisecond * 10):
		}
	}
}
