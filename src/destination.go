package main

import (
	"context"
	"fmt"
	"sync"
)

func NewDestination(name string, dc DestinationConfig) *Destination {
	return &Destination{
		name:   name,
		dc:     dc,
		Logger: NewLogger(dc.LogLevel, "destination"),
	}
}

type Destination struct {
	name   string
	dc     DestinationConfig
	Logger *Logger
}

func (dest *Destination) Start(ctx context.Context, cancel context.CancelFunc) error {
	metricCh := make(chan MetricUnit)
	moCh := make(chan MysqlOperation)
	defer close(metricCh)
	defer close(moCh)

	dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/?%s", dest.dc.User, dest.dc.Password, dest.dc.Host, dest.dc.Port, dest.dc.Params)

	gtidSets, err := NewGtidSets(dest.dc.LogLevel, dest.dc.HJDB.Addr, dest.dc.HJDB.DB, dest.dc.GtidSets)
	if err != nil {
		return err
	}

	metricDirector := NewMetricDirector(dest.dc.LogLevel, "destination", metricCh)
	tcpClient := NewTCPClient(dest.dc.LogLevel, dest.dc.TCPAddr, metricCh)
	mysqlClient, err := NewMysqlClient(dest.dc.LogLevel, dns)
	if err != nil {
		dest.Logger.Error("NewMysqlClient error: " + err.Error())
		return err
	}
	defer mysqlClient.Close()

	mysqlApplier := NewMysqlApplier(dest.dc.LogLevel, gtidSets, mysqlClient, metricCh)

	dest.Logger.Info("Destination start from gtidsets '" + gtidSets.GtidSetsRangeStr + "'")

	mdCtx, mdCancel := context.WithCancel(context.Background())
	defer mdCancel()

	var wg0 sync.WaitGroup
	wg0.Add(1)
	go func() {
		defer wg0.Done()
		metricDirector.Logger.Info("started.")
		metricDirector.Start(mdCtx, "0.0.0.0:9092")
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
		tcpClient.Start(ctx, moCh, gtidSets.GtidSetsRangeStr)
		tcpClient.Logger.Info("stopped.")
		cancel()
	}()
	wg.Wait()

	mdCancel()
	wg0.Wait()

	dest.Logger.Info("stopped")

	return nil
}
