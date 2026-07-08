package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// HTTP Metrics
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed.",
		},
		[]string{"method", "endpoint", "status"},
	)

	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	HttpRequestsInProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_progress",
			Help: "Number of HTTP requests currently in progress.",
		},
		[]string{"method", "endpoint"},
	)

	HttpRequestErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_errors_total",
			Help: "Total number of HTTP request errors.",
		},
		[]string{"method", "endpoint", "status"},
	)

	// Sandbox Metrics
	SandboxCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sandbox_created_total",
			Help: "Total number of sandboxes created.",
		},
	)

	SandboxDeletedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sandbox_deleted_total",
			Help: "Total number of sandboxes deleted.",
		},
	)

	// VSI Connectivity Metrics
	VsiConnectionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vsi_connection_total",
			Help: "Total number of VSI connection attempts.",
		},
		[]string{"status"}, // success, failure, timeout
	)

	VsiConnectionFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vsi_connection_failed_total",
			Help: "Total number of VSI connection failures.",
		},
		[]string{"reason"}, // delay, failure, timeout
	)

	VsiConnectionDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "vsi_connection_duration_seconds",
			Help:    "Latency of VSI connections in seconds.",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 15},
		},
	)

	// Database Metrics
	DbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2},
		},
		[]string{"query_type", "table"}, // query_type: SELECT, INSERT, UPDATE, DELETE
	)

	DbQueryErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Total number of database query errors.",
		},
		[]string{"query_type", "table"},
	)

	ActiveDatabaseConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_database_connections",
			Help: "Number of active database connections in the connection pool.",
		},
	)
)

func RegisterMetrics() {
	prometheus.MustRegister(HttpRequestsTotal)
	prometheus.MustRegister(HttpRequestDuration)
	prometheus.MustRegister(HttpRequestsInProgress)
	prometheus.MustRegister(HttpRequestErrorsTotal)
	prometheus.MustRegister(SandboxCreatedTotal)
	prometheus.MustRegister(SandboxDeletedTotal)
	prometheus.MustRegister(VsiConnectionTotal)
	prometheus.MustRegister(VsiConnectionFailedTotal)
	prometheus.MustRegister(VsiConnectionDuration)
	prometheus.MustRegister(DbQueryDuration)
	prometheus.MustRegister(DbQueryErrorsTotal)
	prometheus.MustRegister(ActiveDatabaseConnections)
}
