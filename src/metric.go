package main

import (
	"context"
	"sort"
	"time"
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
	MetricExtractOperations

	MetricTCPServerOperations
	MetricTCPServerOutgoing

	MetricReplDelay
	MetricReplDMLInsertTimes
	MetricReplDMLUpdateTimes
	MetricReplDMLDeleteTimes
	MetricReplDDLDatabaseTimes
	MetricReplDDLTableTimes
	MetricReplTrx
	MetricApplierOperations
	MetricTCPClientOperations
)

// type MetricDestination struct {
// 	Delay            uint64 `json:"delay"`
// 	DMLInsertTimes   uint64 `json:"dml_insert_times"`
// 	DMLUpdateTimes   uint64 `json:"dml_update_times"`
// 	DMLDeleteTimes   uint64 `json:"dml_delete_times"`
// 	DDLDatabaseTimes uint64 `json:"ddl_database_times"`
// 	DDLTableTimes    uint64 `json:"ddl_table_times"`
// 	Trx              uint64 `json:"trx"`
// 	MergeTrx         uint64 `json:"merge_trx"`
// }

type MetricTCPServer struct {
	Operations uint `json:"operations"`
	Outgoing   uint `json:"outgoing"`
}

type MetricTCPClient struct {
	Operations uint `json:"operations"`
	Incoming   uint `json:"incoming"`
}

// type MetricReplication struct {
// 	Delay            uint64 `json:"delay"`
// 	DMLInsertTimes   uint64 `json:"dml_insert_times"`
// 	DMLUpdateTimes   uint64 `json:"dml_update_times"`
// 	DMLDeleteTimes   uint64 `json:"dml_delete_times"`
// 	DDLDatabaseTimes uint64 `json:"ddl_database_times"`
// 	DDLTableTimes    uint64 `json:"ddl_table_times"`
// }

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
	Name     string
	Logger   *Logger
	hjdb     *HJDB
	metrics  map[string]*PrometheusMetric
	metricCh <-chan MetricUnit

	change bool
}

func NewMetricDirector(logLevel int, hjdb *HJDB, name string, metricCh <-chan MetricUnit) *MetricDirector {
	return &MetricDirector{
		Name:     name,
		Logger:   NewLogger(logLevel, "metric director"),
		hjdb:     hjdb,
		metrics:  make(map[string]*PrometheusMetric),
		metricCh: metricCh,
		change:   false,
	}
}

func (md *MetricDirector) inc(name string, value uint) {
	if _, exists := md.metrics[name]; exists {
		md.metrics[name].MetricValue += value
	} else {
		md.metrics[name] = &PrometheusMetric{MetricType: "counter", MetricValue: 0}
		md.metrics[name].MetricName = name
		md.metrics[name].MetricValue = value
	}
	md.change = true
}

func (md *MetricDirector) set(name string, value uint) {
	if _, exists := md.metrics[name]; exists {
		md.metrics[name].MetricValue = value
	} else {
		md.metrics[name] = &PrometheusMetric{MetricType: "gauge", MetricValue: 0}
		md.metrics[name].MetricValue = value
		md.metrics[name].MetricName = name
	}
	md.change = true
}

func (md *MetricDirector) push() {
	var metricsSlice []*PrometheusMetric
	for _, metric := range md.metrics {
		metricsSlice = append(metricsSlice, metric)
	}

	sort.Slice(metricsSlice, func(i, j int) bool {
		return metricsSlice[i].MetricName < metricsSlice[j].MetricName
	})

	if err := md.hjdb.Update("memory", md.Name, metricsSlice); err != nil {
		md.Logger.Error("push error " + err.Error())
	}
}

func (md *MetricDirector) Start(ctx context.Context) {
	for {
		select {
		case <-time.After(time.Millisecond * 100):
		case <-ctx.Done():
			md.Logger.Info("ctx done signal received.")
			return
		case metric := <-md.metricCh:
			switch metric.Name {
			case MetricDestDMLInsertTimes:
				md.inc("dml_insert_times", metric.Value)
			case MetricDestDMLDeleteTimes:
				md.inc("dml_delete_times", metric.Value)
			case MetricDestDMLUpdateTimes:
				md.inc("dml_delete_times", metric.Value)
			case MetricDestTrx:
				md.inc("trx", metric.Value)
			case MetricDestMergeTrx:
				md.inc("merge_trx", metric.Value)
			case MetricDestDDLDatabaseTimes:
				md.inc("ddl_database_times", metric.Value)
			case MetricDestDDLTableTimes:
				md.inc("ddl_table_times", metric.Value)
			case MetricDestDelay:
				md.set("delay", metric.Value)
			case MetricTCPClientOperations:
				md.inc("tcp_client_operations", metric.Value)
			case MetricApplierOperations:
				md.inc("applier_operations", metric.Value)

			case MetricReplTrx:
				md.inc("trx", metric.Value)
			case MetricExtractOperations:
				md.inc("extract_operations", metric.Value)
			case MetricTCPServerOutgoing:
				md.inc("tcp_server_outgoing_bytes", metric.Value)
			case MetricTCPServerOperations:
				md.inc("tcp_server_operations", metric.Value)
			}

			if md.change {
				md.push()
				md.change = false
			}
		}
	}
}
