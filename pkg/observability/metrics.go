package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP Metrics
var (
	// HTTPRequestsTotal counts total HTTP requests by method, endpoint, and status
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTPRequestDuration measures request duration in seconds
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mocktool_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	// HTTPRequestSize measures request size in bytes
	HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mocktool_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "endpoint"},
	)

	// HTTPResponseSize measures response size in bytes
	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mocktool_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"method", "endpoint"},
	)

	// HTTPErrorsTotal counts HTTP errors by type
	HTTPErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_http_errors_total",
			Help: "Total number of HTTP errors",
		},
		[]string{"method", "endpoint", "error_type"},
	)
)

// Database Metrics
var (
	// DBOperationsTotal counts database operations by type and status
	DBOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_db_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "collection", "status"},
	)

	// DBOperationDuration measures database operation duration
	DBOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mocktool_db_operation_duration_seconds",
			Help:    "Database operation duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation", "collection"},
	)

	// DBConnectionPoolSize tracks connection pool metrics
	DBConnectionPoolSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mocktool_db_connection_pool_size",
			Help: "Current database connection pool size",
		},
		[]string{"state"}, // available, in_use
	)

	// DBErrorsTotal counts database errors
	DBErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_db_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation", "collection", "error_type"},
	)
)

// Business Logic Metrics
var (
	// MockAPILookupTotal counts mock API lookups
	MockAPILookupTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_mock_api_lookup_total",
			Help: "Total number of mock API lookups",
		},
		[]string{"feature", "scenario", "status"},
	)

	// MockAPILookupDuration measures mock API lookup duration
	MockAPILookupDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "mocktool_mock_api_lookup_duration_seconds",
			Help:    "Duration of mock API lookups in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5},
		},
	)

	// MockAPICacheHits counts cache hits vs misses
	MockAPICacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_mock_api_cache_hits_total",
			Help: "Total number of cache hits/misses",
		},
		[]string{"result"}, // hit, miss
	)

	// ActiveScenariosTotal counts active scenarios by feature
	ActiveScenariosTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mocktool_active_scenarios_total",
			Help: "Number of active scenarios per feature",
		},
		[]string{"feature"},
	)

	// LoadTestExecutionsTotal counts load test executions
	LoadTestExecutionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_loadtest_executions_total",
			Help: "Total number of load test executions",
		},
		[]string{"scenario", "status"},
	)

	// LoadTestDuration measures load test execution time
	LoadTestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mocktool_loadtest_duration_seconds",
			Help:    "Load test execution duration in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"scenario"},
	)

	// LoadTestRequestsTotal counts requests made during load tests
	LoadTestRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_loadtest_requests_total",
			Help: "Total number of requests in load tests",
		},
		[]string{"scenario", "status"},
	)
)

// Security Metrics
var (
	// SecurityValidationFailures counts validation failures
	SecurityValidationFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_security_validation_failures_total",
			Help: "Total number of security validation failures",
		},
		[]string{"field", "validation_type"},
	)

	// SecurityHeadersSanitized counts sanitized headers
	SecurityHeadersSanitized = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_security_headers_sanitized_total",
			Help: "Total number of headers sanitized",
		},
		[]string{"header_name", "reason"},
	)

	// SecurityInjectionAttempts counts potential injection attempts
	SecurityInjectionAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mocktool_security_injection_attempts_total",
			Help: "Total number of potential injection attempts detected",
		},
		[]string{"attack_type"},
	)
)

// System Metrics
var (
	// AppInfo provides application version and build info
	AppInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mocktool_app_info",
			Help: "Application information (version, build time, etc.)",
		},
		[]string{"version", "go_version"},
	)

	// UptimeSeconds tracks application uptime
	UptimeSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mocktool_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)
)
