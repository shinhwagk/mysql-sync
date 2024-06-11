package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// 假设 JSON 数据结构如 {"metric1": 100, "metric2": 200}
type Metrics struct {
	Data map[string]int `json:"data"`
}

var (
	registry = prometheus.NewRegistry()              // 创建 Prometheus 注册表
	jsonURL  = "http://your-json-source.com/metrics" // JSON 数据源 URL
)

func fetchAndUpdateMetrics() {
	resp, err := http.Get(jsonURL)
	if err != nil {
		fmt.Println("Error fetching JSON:", err)
		return
	}
	defer resp.Body.Close()

	var metrics Metrics
	err = json.NewDecoder(resp.Body).Decode(&metrics)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	for key, value := range metrics.Data {
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: key,
			Help: "Metric " + key,
		})
		registry.MustRegister(gauge)
		gauge.Set(float64(value))
	}
}

func metricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 在返回指标之前，先获取并更新指标
		fetchAndUpdateMetrics()
		// 使用 promhttp.HandlerFor 返回注册表中的指标
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	})
}

func main() {
	http.Handle("/metrics", metricsHandler())
	http.ListenAndServe(":9090", nil)
}
