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

type MetricUnit struct {
	Name  uint
	Value uint
}

type PrometheusMetric struct {
	MetricName   string
	MetricType   string // gauge, counter
	MetricValue  uint
	MetricLabels map[string]string

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
}

func NewMetricDirector(logLevel int, subsystem string, metricCh <-chan MetricUnit) *MetricDirector {
	return &MetricDirector{
		Name:     "destination",
		Logger:   NewLogger(logLevel, "metric director"),
		metrics:  make(map[string]*PrometheusMetric),
		metricCh: metricCh,

		PromNamespace: "mysqlsync",
		PromSubsystem: subsystem,
		PromRegistry:  prometheus.NewRegistry(),
	}
}

func (md *MetricDirector) inc(name string, value uint) {
	if _, exists := md.metrics[name]; !exists {
		md.registerMetric(name, "counter")
	}
	metric, _ := md.metrics[name]
	counter, _ := metric.PromCollector.(prometheus.Counter)
	counter.Add(float64(value))

}

func (md *MetricDirector) set(name string, value uint) {
	if _, exists := md.metrics[name]; !exists {
		md.registerMetric(name, "gauge")
	}
	metric, _ := md.metrics[name]
	gauge, _ := metric.PromCollector.(prometheus.Gauge)
	gauge.Set(float64(value))
}

func (md *MetricDirector) Start(ctx context.Context, addr string) {
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

			case MetricReplDelay:
				md.set("delay", metric.Value)
			case MetricReplTrx:
				md.inc("trx", metric.Value)
			case MetricExtractOperations:
				md.inc("extract_operations", metric.Value)
			case MetricTCPServerOutgoing:
				md.inc("tcp_server_outgoing_bytes", metric.Value)
			case MetricTCPServerOperations:
				md.inc("tcp_server_operations", metric.Value)
			}
		}
	}
}

func (md *MetricDirector) registerMetric(name string, metricType string) {
	if _, exists := md.metrics[name]; !exists {
		var collector prometheus.Collector
		if metricType == "gauge" {
			collector = prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: md.PromNamespace,
				Subsystem: md.PromSubsystem,
				Name:      name,
				Help:      "A dynamically created gauge",
			})
		} else if metricType == "counter" {
			collector = prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: md.PromNamespace,
				Subsystem: md.PromSubsystem,
				Name:      name,
				Help:      "A dynamically created counter",
			})
		}

		md.PromRegistry.MustRegister(collector)
		md.metrics[name] = &PrometheusMetric{
			MetricName:    name,
			MetricType:    metricType,
			PromCollector: collector,
		}
		md.Logger.Info(fmt.Sprintln("Metric registered: %s, Type: %s", name, metricType))
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
			// md.Logger.Printf("HTTP server Shutdown: %v", err)
		}
	}()

	md.Logger.Info(fmt.Sprintf("Prometheus metrics are being served at %s/metrics", addr))
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// md.Logger.Printf("HTTP server ListenAndServe: %v", err)
	}
}
