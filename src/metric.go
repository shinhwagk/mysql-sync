package main

import (
	"context"
	"fmt"
)

const (
	MetricDestDelay uint = iota
	MetricDestDMLInsertTimes
	MetricDestDMLUpdateTimes
	MetricDestDMLDeleteTimes
	MetricDestDDLDatabaseTimes
	MetricDestDDLTableTimes
	MetricDestTrx
	MetricDestMergeTrx

	MetricReplDelay
	MetricReplDMLInsertTimes
	MetricReplDMLUpdateTimes
	MetricReplDMLDeleteTimes
	MetricReplDDLDatabaseTimes
	MetricReplDDLTableTimes
	MetricReplTrx
)

type MetricDestination struct {
	Delay            uint64 `json:"delay"`
	DMLInsertTimes   uint64 `json:"dml_insert_times"`
	DMLUpdateTimes   uint64 `json:"dml_update_times"`
	DMLDeleteTimes   uint64 `json:"dml_delete_times"`
	DDLDatabaseTimes uint64 `json:"ddl_database_times"`
	DDLTableTimes    uint64 `json:"ddl_table_times"`
	Trx              uint64 `json:"trx"`
	MergeTrx         uint64 `json:"merge_trx"`
}

type MetricTCPServer struct {
	Operations uint `json:"operations"`
	Outgoing   uint `json:"outgoing"`
}

type MetricTCPClient struct {
	Operations uint `json:"operations"`
	Incoming   uint `json:"incoming"`
}

type MetricReplication struct {
	Delay            uint64 `json:"delay"`
	DMLInsertTimes   uint64 `json:"dml_insert_times"`
	DMLUpdateTimes   uint64 `json:"dml_update_times"`
	DMLDeleteTimes   uint64 `json:"dml_delete_times"`
	DDLDatabaseTimes uint64 `json:"ddl_database_times"`
	DDLTableTimes    uint64 `json:"ddl_table_times"`
}

type MetricUnit struct {
	Name  uint
	Value uint
}

type PrometheusMetric struct {
	MetricName   string
	MetricType   string
	MetricValue  uint
	MetricLabels map[string]string
}

type MetricDirector struct {
	Name    string
	logger  *Logger
	hjdb    *HJDB
	metrics map[string]*PrometheusMetric
}

func NewMetricDirector(logLevel int, hjdb *HJDB, name string) *MetricDirector {
	return &MetricDirector{
		Name:    name,
		logger:  NewLogger(logLevel, "metric director"),
		hjdb:    hjdb,
		metrics: make(map[string]*PrometheusMetric),
	}
}

func (md *MetricDirector) inc(name string) {
	if _, exists := md.metrics[name]; exists {
		md.metrics[name].MetricValue += 1
	} else {
		md.metrics[name] = &PrometheusMetric{MetricType: "counter", MetricValue: 0}
		md.metrics[name].MetricName = name
		md.metrics[name].MetricValue = 1
	}
}

func (md *MetricDirector) set(name string, value uint) {
	if _, exists := md.metrics[name]; exists {
		md.metrics[name].MetricValue = value
	} else {
		md.metrics[name] = &PrometheusMetric{MetricType: "gauge", MetricValue: 0}
		md.metrics[name].MetricValue = value
		md.metrics[name].MetricName = name
	}
}

func (md *MetricDirector) push() {
	var metricsSlice []*PrometheusMetric
	for _, metric := range md.metrics {
		metricsSlice = append(metricsSlice, metric)
	}
	if err := md.hjdb.Update("memory", md.Name, metricsSlice); err != nil {
		md.logger.Error("push error " + err.Error())
	}
}

func (md *MetricDirector) Start(ctx context.Context, metricCh <-chan MetricUnit) {
	md.logger.Info("started.")

	for metric := range metricCh {
		fmt.Println("metric.. xxxxx")
		select {
		case <-ctx.Done():
			// return ctx.Err()
		default:
			switch metric.Name {
			case MetricDestDMLInsertTimes:
				md.inc("dml_insert_times")
			case MetricDestDMLDeleteTimes:
				md.inc("dml_delete_times")
			case MetricDestDMLUpdateTimes:
				md.inc("dml_delete_times")
			case MetricDestTrx:
				md.inc("trx")
			case MetricDestMergeTrx:
				md.inc("merge_trx")
			case MetricDestDDLDatabaseTimes:
				md.inc("ddl_database_times")
			case MetricDestDDLTableTimes:
				md.inc("ddl_table_times")
			case MetricDestDelay:
				md.set("delay", metric.Value)

			case MetricReplTrx:
				md.inc("trx")
			}

		}
		md.push()
	}
}
