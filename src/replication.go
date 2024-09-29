package main

import (
	"context"
)

func NewReplication(msc MysqlSyncConfig) *Replication {
	return &Replication{
		msc:    msc,
		Logger: NewLogger(parseLogLevel(msc.LogLevel), "replication"),
	}
}

type Replication struct {
	msc    MysqlSyncConfig
	Logger *Logger
}

func (repl *Replication) Start(ctx context.Context, cancel context.CancelFunc) {
	repl.Logger.Info("Started.")
	defer repl.Logger.Info("Closed.")

	metricCh := make(chan MetricUnit)
	defer close(metricCh)

	destStartGtidSetsStrCh := make(chan DestStartGtidSetsRangeStr)
	defer close(destStartGtidSetsStrCh)

	cacheSize := max(repl.msc.Replication.Settings.CacheSize, 1000)
	repl.Logger.Info("Settings cache size: %d", cacheSize)
	moCh := make(chan MysqlOperation, cacheSize)
	defer close(moCh)

	if len(repl.msc.Destination.Destinations) == 0 {
		repl.Logger.Error("dest number is 0.")
		return
	}

	destNames := []string{}
	for destName := range repl.msc.Destination.Destinations {
		destNames = append(destNames, destName)
	}

	ctxMd, cancelMd := context.WithCancel(context.Background())
	ctxTs, cancelTs := context.WithCancel(context.Background())
	ctxEx, cancelEx := context.WithCancel(context.Background())

	go func() {
		defer cancelMd()
		defer cancel()
		promExportPort := 9092
		if repl.msc.Replication.Prometheus.ExportPort > 0 {
			promExportPort = repl.msc.Replication.Prometheus.ExportPort
		}
		metricDirector := NewMetricReplDirector(repl.Logger.Level, promExportPort, "replication", repl.msc.Replication.Name, metricCh)
		metricDirector.Start(ctx)
	}()

	go func() {
		defer cancelTs()
		defer cancel()
		tcpServer := NewTCPServer(repl.Logger.Level, repl.msc.Replication.TCPAddr, destNames, moCh, metricCh, destStartGtidSetsStrCh, repl.msc.Replication.Transfer.MaxTimeMs)
		tcpServer.Start(ctx)
	}()

	go func() {
		defer cancelEx()
		defer cancel()
		destGtidSetss := make(map[string]map[string]uint)
		gtidSetss := make([]map[string]uint, 0, len(repl.msc.Destination.Destinations))
		for i := 0; i < len(destNames); i++ {
			select {
			case <-ctx.Done():
				return
			case gtidSetsStr := <-destStartGtidSetsStrCh:
				if gss, err := GetGtidSetsMapFromGtidSetsRangeStr(gtidSetsStr.GtidSetsStr); err != nil {
					repl.Logger.Error("Convert gtidsets map to str error: %s.", err)
					return
				} else {
					if _, exists := destGtidSetss[gtidSetsStr.DestName]; exists {
						repl.Logger.Error("Dest '%s' gtidset exist '%#v'.", gtidSetsStr.DestName, gss)
						return
					} else {
						destGtidSetss[gtidSetsStr.DestName] = gss
						gtidSetss = append(gtidSetss, gss)
					}
				}
				repl.Logger.Info("Receive init client info: %s@%s", gtidSetsStr.DestName, gtidSetsStr.GtidSetsStr)
			}
		}
		initGtidSetsRangeStr := GetGtidSetsRangeStrFromGtidSetsMap(MergeGtidSetss(gtidSetss))
		repl.Logger.Info("Init gtidsets range string:'%s'", initGtidSetsRangeStr)

		extract := NewBinlogExtract(repl.Logger.Level, repl.msc.Replication, initGtidSetsRangeStr, moCh, metricCh)
		extract.Start(ctx)
	}()

	<-ctxMd.Done()
Loop:
	for {
		select {
		case <-metricCh:
		case <-ctxTs.Done():
			break Loop
			// case <-time.After(time.Millisecond * 10):
		}
	}

	for {
		select {
		case <-moCh:
		case <-metricCh:
		case <-ctxEx.Done():
			return
			// case <-time.After(time.Millisecond * 10):
		}
	}
}
