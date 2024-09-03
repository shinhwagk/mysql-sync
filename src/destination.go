package main

import (
	"context"
)

func NewDestination(msc MysqlSyncConfig, destName string) *Destination {
	return &Destination{
		msc:    msc,
		Name:   destName,
		Logger: NewLogger(parseLogLevel(msc.LogLevel), "destination"),
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

	cacheSize := max(dest.msc.Destination.CacheSize, 1000)
	moCh := make(chan MysqlOperation, cacheSize)
	defer close(moCh)

	// gtidsets must first init.
	ckpt := NewCheckpoint(dest.Logger.Level, consulAddr, replName, dest.Name)
	err := ckpt.InitStartupGtidSetsMap(destConf.Sync.InitGtidSetsRangeStr)
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
		if destConf.Prometheus.ExportPort > 0 {
			promExportPort = destConf.Prometheus.ExportPort
		}
		metricDirector := NewMetricDestDirector(dest.Logger.Level, promExportPort, "destination", replName, dest.Name, metricCh)
		metricDirector.Start(ctx)
	}()

	go func() {
		defer cancelMa()
		defer cancel()
		replicateFilter := NewReplicateFilter(destConf.Sync.Replicate)
		mysqlClient, err := NewMysqlClient(dest.Logger.Level, destConf.Mysql)
		if err != nil {
			dest.Logger.Error("NewMysqlClient: %s.", err)
			return
		}
		defer mysqlClient.Close()
		mysqlApplier := NewMysqlApplier(dest.Logger.Level, ckpt, mysqlClient, replicateFilter, metricCh)
		mysqlApplier.Start(ctx, moCh)
	}()

	go func() {
		defer cancelTc()
		defer cancel()
		tcpClient, err := NewTCPClient(dest.Logger.Level, tcpAddr, dest.Name, moCh, metricCh, GetGtidSetsRangeStrFromGtidSetsMap(ckpt.GtidSetsMap))
		if err != nil {
			dest.Logger.Error("NewTCPClient: %s.", err)
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
			// case <-time.After(time.Millisecond * 10):
		}
	}

	for {
		select {
		case <-moCh:
		case <-metricCh:
		case <-ctxTc.Done():
			return
			// case <-time.After(time.Millisecond * 10):
		}
	}
}
