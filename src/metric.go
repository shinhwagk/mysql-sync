package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	MetricTCPServerSendOperations
	MetricTCPServerOutgoing

	MetricReplDelay
	MetricReplDMLInsertTimes
	MetricReplDMLUpdateTimes
	MetricReplDMLDeleteTimes
	MetricReplDDLDatabaseTimes
	MetricReplDDLTableTimes
	MetricReplTrx
	MetricApplierOperations
	MetricTCPClientReceiveOperations
	MetricTCPClientRequestOperations
)

type MetricUnit struct {
	Name  uint
	Value uint
	Dest  *string
}

type PrometheusMetric struct {
	MetricName   string
	MetricType   string // gauge, counter
	MetricValue  uint
	MetricLabels []string

	PromCollector prometheus.Collector
}

type MetricDirector struct {
	Name     string
	Logger   *Logger
	metrics  map[string]*PrometheusMetric
	metricCh <-chan MetricUnit

	// prometheus
	PromNamespace string
	PromSubsystem string
	PromRegistry  *prometheus.Registry

	ReplName string
	DestName *string
}

func NewMetricReplDirector(logLevel int, subsystem string, replName string, metricCh <-chan MetricUnit) *MetricDirector {
	return &MetricDirector{
		Name:     "destination",
		Logger:   NewLogger(logLevel, "metric director"),
		metrics:  make(map[string]*PrometheusMetric),
		metricCh: metricCh,

		PromNamespace: "mysqlsync",
		PromSubsystem: subsystem,
		PromRegistry:  prometheus.NewRegistry(),

		ReplName: replName,
	}
}

func NewMetricDestDirector(logLevel int, subsystem string, replName string, destName string, metricCh <-chan MetricUnit) *MetricDirector {
	return &MetricDirector{
		Name:     "destination",
		Logger:   NewLogger(logLevel, "metric director"),
		metrics:  make(map[string]*PrometheusMetric),
		metricCh: metricCh,

		PromNamespace: "mysqlsync",
		PromSubsystem: subsystem,
		PromRegistry:  prometheus.NewRegistry(),

		ReplName: replName,
		DestName: &destName,
	}
}

func (md *MetricDirector) inc(name string, value uint, dest *string) {
	if _, exists := md.metrics[name]; !exists {
		labelNames := make([]string, 0, 2)
		labelNames = append(labelNames, "repl")
		if dest != nil {
			labelNames = append(labelNames, "dest")
		}
		md.registerMetric(name, "counter", labelNames)
	}
	var labelValues []string
	labelValues = append(labelValues, md.ReplName)
	if dest != nil {
		labelValues = append(labelValues, *dest)
	}
	metric, _ := md.metrics[name]
	counter, _ := metric.PromCollector.(*prometheus.CounterVec)
	counter.WithLabelValues(labelValues...).Add(float64(value))
}

func (md *MetricDirector) set(name string, value uint, dest *string) {
	if _, exists := md.metrics[name]; !exists {
		labelNames := make([]string, 0, 2)
		labelNames = append(labelNames, "repl")
		if dest != nil {
			labelNames = append(labelNames, "dest")
		}
		md.registerMetric(name, "gauge", labelNames)
	}
	var labelValues []string
	labelValues = append(labelValues, md.ReplName)
	if dest != nil {
		labelValues = append(labelValues, *dest)
	}
	metric, _ := md.metrics[name]
	gauge, _ := metric.PromCollector.(*prometheus.GaugeVec)
	gauge.WithLabelValues(labelValues...).Set(float64(value))
}

func (md *MetricDirector) Start(ctx context.Context, addr string) {
	md.Logger.Info("Started.")
	defer md.Logger.Info("Closed.")

	go func() {
		md.StartHTTPServer(ctx, addr)
	}()

	for {
		select {
		case <-time.After(time.Millisecond * 100):
		case <-ctx.Done():
			md.Logger.Info("ctx done signal received.")
			return
		case metric := <-md.metricCh:
			switch metric.Name {
			case MetricDestDMLInsertTimes:
				md.inc("dml_insert_times", metric.Value, md.DestName)
			case MetricDestDMLDeleteTimes:
				md.inc("dml_delete_times", metric.Value, md.DestName)
			case MetricDestDMLUpdateTimes:
				md.inc("dml_update_times", metric.Value, md.DestName)
			case MetricDestTrx:
				md.inc("trx", metric.Value, md.DestName)
			case MetricDestMergeTrx:
				md.inc("merge_trx", metric.Value, md.DestName)
			case MetricDestDDLDatabaseTimes:
				md.inc("ddl_database_times", metric.Value, md.DestName)
			case MetricDestDDLTableTimes:
				md.inc("ddl_table_times", metric.Value, md.DestName)
			case MetricDestDelay:
				md.set("delay", metric.Value, md.DestName)
			case MetricTCPClientReceiveOperations:
				md.inc("tcp_client_receive_operations", metric.Value, md.DestName)
			case MetricTCPClientRequestOperations:
				md.inc("tcp_client_request_operations", metric.Value, md.DestName)
			case MetricApplierOperations:
				md.inc("applier_operations", metric.Value, md.DestName)

			case MetricReplDelay:
				md.set("delay", metric.Value, nil)
			case MetricReplTrx:
				md.inc("trx", metric.Value, nil)
			case MetricExtractOperations:
				md.inc("extract_operations", metric.Value, nil)
			case MetricTCPServerOutgoing:
				md.inc("tcp_server_outgoing_bytes", metric.Value, metric.Dest)
			case MetricTCPServerSendOperations:
				md.inc("tcp_server_send_operations", metric.Value, metric.Dest)
			}
		}
	}
}

func (md *MetricDirector) registerMetric(name string, metricType string, labelNames []string) {
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
		md.Logger.Info(fmt.Sprintf("Metric registered: %s, Type: %s", name, metricType))
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
			md.Logger.Error(fmt.Sprintf("HTTP server Shutdown: %s", err.Error()))
		}
	}()

	md.Logger.Info(fmt.Sprintf("Prometheus metrics are being served at %s/metrics", addr))
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		md.Logger.Error(fmt.Sprintf("HTTP server ListenAndServe: %s", err))
	}
}
