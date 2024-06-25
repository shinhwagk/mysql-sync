package main

import (
	"context"
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
	moCh := make(chan MysqlOperation, 1000)

	defer close(metricCh)
	defer close(moCh)

	destNames := []string{}

	destGtidSetss := make([]map[string]uint, len(repl.msc.Destinations))
	for _, dc := range repl.msc.Destinations {
		gss := NewGtidSets(repl.msc.HJDB.Addr, repl.msc.Replication.Name, dc.Name)
		gss.InitStartupGtidSetsMap(dc.InitGtidSetsRangeStr)
		destGtidSetss = append(destGtidSetss, gss.GtidSetsMap)

		destNames = append(destNames, dc.Name)
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
