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
	consulAddr := dest.msc.Consul.Addr
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
	gtidSets := NewGtidSets(destConf.LogLevel, consulAddr, replName, dest.Name)
	err := gtidSets.InitStartupGtidSetsMap(destConf.Sync.InitGtidSetsRangeStr)
	if err != nil {
		return
	}

	ctxMa, cancelMa := context.WithCancel(context.Background())
	ctxTc, cancelTc := context.WithCancel(context.Background())
	ctxMd, cancelMd := context.WithCancel(context.Background())

	go func() {
		defer cancelMd()
		defer cancel()
		promExportPort := 9092
		if destConf.Prometheus != nil {
			if destConf.Prometheus.ExportPort != 0 {
				promExportPort = destConf.Prometheus.ExportPort
			} else {
				dest.Logger.Error("prometheus export port %d.", destConf.Prometheus.ExportPort)
				return
			}
		}
		metricDirector := NewMetricDestDirector(destConf.LogLevel, "destination", replName, dest.Name, metricCh)
		metricDirector.Start(ctx, fmt.Sprintf("0.0.0.0:%d", promExportPort))
	}()

	go func() {
		defer cancelMa()
		defer cancel()
		replicateFilter := NewReplicateFilter(destConf.Sync.Replicate)
		mysqlClient, err := NewMysqlClient(destConf.LogLevel, destConf.Mysql)
		if err != nil {
			dest.Logger.Error("NewMysqlClient: %s", err)
			return
		}
		defer mysqlClient.Close()
		mysqlApplier := NewMysqlApplier(destConf.LogLevel, gtidSets, mysqlClient, replicateFilter, metricCh)
		mysqlApplier.Start(ctx, moCh)

	}()

	go func() {
		defer cancelTc()
		defer cancel()
		tcpClient, err := NewTCPClient(destConf.LogLevel, tcpAddr, dest.Name, moCh, metricCh, GetGtidSetsRangeStrFromGtidSetsMap(gtidSets.GtidSetsMap))
		if err != nil {
			dest.Logger.Error("NewTCPClient: %s", err)
			return
		}
		tcpClient.Start(ctx)
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
