package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	MetricDestCheckpointTimestamp uint = iota
	MetricDestApplierTimestamp
	MetricDestDMLInsertTimes
	MetricDestDMLUpdateTimes
	MetricDestDMLDeleteTimes
	MetricDestDMLSkipInsertTimes
	MetricDestDMLSkipUpdateTimes
	MetricDestDMLSkipDeleteTimes
	MetricDestDDLDatabaseTimes
	MetricDestDDLTableTimes
	MetricDestDDLSkipDatabaseTimes
	MetricDestDDLSkipTableTimes
	MetricDestTrx
	MetricDestMergeTrx
	MetricDestApplierOperations
	MetricDestApplierSkipOperations

	MetricTCPServerSendOperations
	MetricTCPServerOutgoing

	MetricReplExtractorTimestamp
	MetricReplDMLInsertTimes
	MetricReplDMLUpdateTimes
	MetricReplDMLDeleteTimes
	MetricReplDDLDatabaseTimes
	MetricReplDDLTableTimes
	MetricReplExtractorOperations
	MetricTCPServerSendDelay
	MetricReplTrx
	// MetricDestApplierSkipOperations
	MetricTCPClientReceiveOperations
)

type MetricUnit struct {
	Name      uint
	Value     uint
	LabelPair map[string]string
}

type PrometheusMetric struct {
	MetricName    string
	MetricType    string // gauge, counter
	MetricValue   uint
	MetricLabels  []string
	PromCollector prometheus.Collector
}

type MetricDirector struct {
	Name           string
	Logger         *Logger
	metrics        map[string]*PrometheusMetric
	metricCh       <-chan MetricUnit
	PromExportPort int
	PromNamespace  string
	PromSubsystem  string
	PromRegistry   *prometheus.Registry
	ReplName       string
	DestName       *string
}

func NewMetricReplDirector(logLevel int, promExportPort int, subsystem string, replName string, metricCh <-chan MetricUnit) *MetricDirector {
	return &MetricDirector{
		Name:           "replication",
		Logger:         NewLogger(logLevel, "metric-director"),
		metrics:        make(map[string]*PrometheusMetric),
		metricCh:       metricCh,
		PromExportPort: promExportPort,
		PromNamespace:  "mysqlsync",
		PromSubsystem:  subsystem,
		PromRegistry:   prometheus.NewRegistry(),
		ReplName:       replName,
	}
}

func NewMetricDestDirector(logLevel int, promExportPort int, subsystem string, replName string, destName string, metricCh <-chan MetricUnit) *MetricDirector {
	return &MetricDirector{
		Name:           "destination",
		Logger:         NewLogger(logLevel, "metric director"),
		metrics:        make(map[string]*PrometheusMetric),
		metricCh:       metricCh,
		PromExportPort: promExportPort,
		PromNamespace:  "mysqlsync",
		PromSubsystem:  subsystem,
		PromRegistry:   prometheus.NewRegistry(),
		ReplName:       replName,
		DestName:       &destName,
	}
}

func (md *MetricDirector) prepareMetric(metricType, name string, labelPair map[string]string) ([]string, []string) {
	keys := make([]string, 0, len(labelPair))
	for key := range labelPair {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	labelNames := []string{"repl"}
	if md.DestName != nil {
		labelNames = append(labelNames, "dest")
	}
	labelNames = append(labelNames, keys...)

	if _, exists := md.metrics[name]; !exists {
		md.registerMetric(metricType, name, labelNames)
	}

	labelValues := []string{md.ReplName}
	if md.DestName != nil {
		labelValues = append(labelValues, *md.DestName)
	}
	for _, key := range keys {
		labelValues = append(labelValues, labelPair[key])
	}

	return labelNames, labelValues
}

func (md *MetricDirector) inc(name string, value uint, labelPair map[string]string) {
	_, labelValues := md.prepareMetric("counter", name, labelPair)
	metric := md.metrics[name]
	counter, _ := metric.PromCollector.(*prometheus.CounterVec)
	counter.WithLabelValues(labelValues...).Add(float64(value))
}

func (md *MetricDirector) set(name string, value uint, labelPair map[string]string) {
	_, labelValues := md.prepareMetric("gauge", name, labelPair)
	metric := md.metrics[name]
	gauge, _ := metric.PromCollector.(*prometheus.GaugeVec)
	gauge.WithLabelValues(labelValues...).Set(float64(value))
}

func (md *MetricDirector) Start(ctx context.Context) {
	md.Logger.Info("Started.")
	defer md.Logger.Info("Closed.")

	go func() {
		md.StartHTTPServer(ctx, fmt.Sprintf("0.0.0.0:%d", md.PromExportPort))
	}()

	for {
		select {
		// case <-time.After(time.Millisecond * 100):
		case <-ctx.Done():
			md.Logger.Info("ctx done signal received.")
			return
		case metric := <-md.metricCh:
			switch metric.Name {
			case MetricDestDMLInsertTimes:
				md.inc("dml_insert_times", metric.Value, metric.LabelPair)
			case MetricDestDMLDeleteTimes:
				md.inc("dml_delete_times", metric.Value, metric.LabelPair)
			case MetricDestDMLUpdateTimes:
				md.inc("dml_update_times", metric.Value, metric.LabelPair)
			case MetricDestDMLSkipInsertTimes:
				md.inc("dml_skip_insert_times", metric.Value, metric.LabelPair)
			case MetricDestDMLSkipDeleteTimes:
				md.inc("dml_skip_delete_times", metric.Value, metric.LabelPair)
			case MetricDestDMLSkipUpdateTimes:
				md.inc("dml_skip_update_times", metric.Value, metric.LabelPair)
			case MetricDestTrx:
				md.inc("trx", metric.Value, metric.LabelPair)
			case MetricDestMergeTrx:
				md.inc("merge_trx", metric.Value, metric.LabelPair)
			case MetricDestDDLDatabaseTimes:
				md.inc("ddl_database_times", metric.Value, metric.LabelPair)
			case MetricDestDDLTableTimes:
				md.inc("ddl_table_times", metric.Value, metric.LabelPair)
			case MetricDestDDLSkipDatabaseTimes:
				md.inc("ddl_skip_database_times", metric.Value, metric.LabelPair)
			case MetricDestDDLSkipTableTimes:
				md.inc("ddl_skip_table_times", metric.Value, metric.LabelPair)
			case MetricDestCheckpointTimestamp:
				md.set("ckeckpoint_timestamp", metric.Value, metric.LabelPair)
			case MetricDestApplierTimestamp:
				md.set("applier_timestamp", metric.Value, metric.LabelPair)
			case MetricTCPClientReceiveOperations:
				md.inc("tcp_client_receive_operations", metric.Value, metric.LabelPair)
			case MetricDestApplierOperations:
				md.inc("applier_operations", metric.Value, metric.LabelPair)
			case MetricDestApplierSkipOperations:
				md.inc("applier_skip_operations", metric.Value, metric.LabelPair)
			// case MetricDestApplierSkipOperations:
			// 	md.inc("applier_skip_operations", metric.Value, metric.LabelPair)

			case MetricReplExtractorTimestamp:
				md.set("extractor_timestamp", metric.Value, metric.LabelPair)
			case MetricTCPServerSendDelay:
				md.set("tcp_server_send_delay", metric.Value, metric.LabelPair)
			case MetricReplTrx:
				md.inc("trx", metric.Value, metric.LabelPair)
			case MetricReplExtractorOperations:
				md.inc("extractor_operations", metric.Value, metric.LabelPair)
			case MetricTCPServerOutgoing:
				md.inc("tcp_server_outgoing_bytes", metric.Value, metric.LabelPair)
			case MetricTCPServerSendOperations:
				md.inc("tcp_server_send_operations", metric.Value, metric.LabelPair)
			case MetricReplDMLInsertTimes:
				md.inc("dml_insert_times", metric.Value, metric.LabelPair)
			case MetricReplDMLDeleteTimes:
				md.inc("dml_delete_times", metric.Value, metric.LabelPair)
			case MetricReplDMLUpdateTimes:
				md.inc("dml_update_times", metric.Value, metric.LabelPair)
			case MetricReplDDLDatabaseTimes:
				md.inc("ddl_database_times", metric.Value, metric.LabelPair)
			case MetricReplDDLTableTimes:
				md.inc("ddl_table_times", metric.Value, metric.LabelPair)
			}
		}
	}
}

func (md *MetricDirector) registerMetric(metricType string, name string, labelNames []string) {
	if _, exists := md.metrics[name]; !exists {
		var collector prometheus.Collector
		if metricType == "gauge" {
			collector = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: md.PromNamespace,
					Subsystem: md.PromSubsystem,
					Name:      name,
					Help:      "A dynamically created gauge",
				},
				labelNames,
			)
		} else if metricType == "counter" {
			collector = prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: md.PromNamespace,
					Subsystem: md.PromSubsystem,
					Name:      name,
					Help:      "A dynamically created counter",
				},
				labelNames,
			)
		}

		md.PromRegistry.MustRegister(collector)
		md.metrics[name] = &PrometheusMetric{
			MetricName:    name,
			MetricType:    metricType,
			PromCollector: collector,
			MetricLabels:  labelNames,
		}
		md.Logger.Info("Metric registered: %s, Type: %s", name, metricType)
	}
}

func (md *MetricDirector) StartHTTPServer(ctx context.Context, addr string) {
	srv := &http.Server{Addr: addr, Handler: nil}

	http.Handle("/metrics", promhttp.HandlerFor(md.PromRegistry, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Mysql Sync Exporter</title></head>
			<body>
			<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>`))
	})

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			md.Logger.Error("HTTP server Shutdown: %s.", err)
		}
	}()

	md.Logger.Info("Prometheus metrics are being served at %s/metrics", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		md.Logger.Error("HTTP server ListenAndServe: %s.", err)
	}
}
