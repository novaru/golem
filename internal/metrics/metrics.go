package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "golem_requests_total",
			Help: "Total number of requests processed by the load balancer",
		},
		[]string{"backend", "method", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "golem_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"backend", "method"},
	)

	ActiveConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "golem_active_connections",
			Help: "Current number of active connections per backend",
		},
		[]string{"backend"},
	)

	BackendHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "golem_backend_health",
			Help: "Backend health status (1 = healthy, 0 = unhealthy)",
		},
		[]string{"backend"},
	)

	LoadBalancerInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "golem_info",
			Help: "Load balancer information",
		},
		[]string{"version", "method"},
	)

	FileOps = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "golem_file_operations_total",
			Help: "Number of file operations by type and backend",
		},
		[]string{"backend", "operation"}, // operation: upload/download/modify/delete
	)

	RequestFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "golem_request_failures_total",
			Help: "Number of failed requests",
		},
		[]string{"backend", "method", "reason"},
	)

	BackendWeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "golem_backend_weight",
			Help: "Weight assigned to each backend for WRR",
		},
		[]string{"backend"},
	)

	QueueDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "golem_queue_duration_seconds",
			Help:    "Time requests spent waiting in queue before being sent to a backend",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"backend"},
	)
)

func UpdateBackendHealth(backend string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	BackendHealth.WithLabelValues(backend).Set(value)
}

func UpdateActiveConnections(backend string, conn float64) {
	ActiveConnections.WithLabelValues(backend).Set(conn)
}

func RecordRequest(backend, method, status string, duration float64) {
	RequestsTotal.WithLabelValues(backend, method, status).Inc()
	RequestDuration.WithLabelValues(backend, method).Observe(duration)
}

func SetLoadBalancerInfo(version, method string) {
	LoadBalancerInfo.WithLabelValues(version, method).Set(1)
}

func FileOpsRequest(backend, operation string) {
	FileOps.WithLabelValues(backend, operation).Inc()
}

func SetBackendWeight(weight float64) {
	BackendWeight.WithLabelValues("app1").Set(weight)
}
