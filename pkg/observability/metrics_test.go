package observability

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestMetricsAreRegistered(t *testing.T) {
	tests := []struct {
		name   string
		metric prometheus.Collector
	}{
		{"HTTPRequestsTotal", HTTPRequestsTotal},
		{"HTTPRequestDuration", HTTPRequestDuration},
		{"HTTPRequestSize", HTTPRequestSize},
		{"HTTPResponseSize", HTTPResponseSize},
		{"HTTPErrorsTotal", HTTPErrorsTotal},
		{"DBOperationsTotal", DBOperationsTotal},
		{"DBOperationDuration", DBOperationDuration},
		{"DBConnectionPoolSize", DBConnectionPoolSize},
		{"DBErrorsTotal", DBErrorsTotal},
		{"MockAPILookupTotal", MockAPILookupTotal},
		{"MockAPILookupDuration", MockAPILookupDuration},
		{"MockAPICacheHits", MockAPICacheHits},
		{"ActiveScenariosTotal", ActiveScenariosTotal},
		{"LoadTestExecutionsTotal", LoadTestExecutionsTotal},
		{"LoadTestDuration", LoadTestDuration},
		{"LoadTestRequestsTotal", LoadTestRequestsTotal},
		{"SecurityValidationFailures", SecurityValidationFailures},
		{"SecurityHeadersSanitized", SecurityHeadersSanitized},
		{"SecurityInjectionAttempts", SecurityInjectionAttempts},
		{"AppInfo", AppInfo},
		{"UptimeSeconds", UptimeSeconds},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.metric, "%s metric should not be nil", tt.name)
		})
	}
}

func TestHTTPMetricsCanBeRecorded(t *testing.T) {
	// Test that we can record HTTP metrics without errors
	HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/test", "200").Inc()
	HTTPRequestDuration.WithLabelValues("GET", "/api/v1/test").Observe(0.5)
	HTTPRequestSize.WithLabelValues("GET", "/api/v1/test").Observe(1024)
	HTTPResponseSize.WithLabelValues("GET", "/api/v1/test").Observe(2048)
	HTTPErrorsTotal.WithLabelValues("GET", "/api/v1/test", "internal_error").Inc()

	// If we get here without panicking, the test passes
	assert.True(t, true)
}

func TestDatabaseMetricsCanBeRecorded(t *testing.T) {
	DBOperationsTotal.WithLabelValues("find", "mockapis", "success").Inc()
	DBOperationDuration.WithLabelValues("find", "mockapis").Observe(0.05)
	DBConnectionPoolSize.WithLabelValues("available").Set(10)
	DBConnectionPoolSize.WithLabelValues("in_use").Set(5)
	DBErrorsTotal.WithLabelValues("find", "mockapis", "not_found").Inc()

	assert.True(t, true)
}

func TestBusinessLogicMetricsCanBeRecorded(t *testing.T) {
	MockAPILookupTotal.WithLabelValues("test-feature", "test-scenario", "found").Inc()
	MockAPILookupDuration.Observe(0.025)
	MockAPICacheHits.WithLabelValues("hit").Inc()
	MockAPICacheHits.WithLabelValues("miss").Inc()
	ActiveScenariosTotal.WithLabelValues("test-feature").Set(5)
	LoadTestExecutionsTotal.WithLabelValues("test-scenario", "success").Inc()
	LoadTestDuration.WithLabelValues("test-scenario").Observe(30.5)
	LoadTestRequestsTotal.WithLabelValues("test-scenario", "success").Add(100)

	assert.True(t, true)
}

func TestSecurityMetricsCanBeRecorded(t *testing.T) {
	SecurityValidationFailures.WithLabelValues("name", "no_spaces").Inc()
	SecurityHeadersSanitized.WithLabelValues("Set-Cookie", "blocked").Inc()
	SecurityInjectionAttempts.WithLabelValues("sql_injection").Inc()
	SecurityInjectionAttempts.WithLabelValues("xss").Inc()

	assert.True(t, true)
}

func TestSystemMetricsCanBeRecorded(t *testing.T) {
	AppInfo.WithLabelValues("1.0.0", "go1.21").Set(1)
	UptimeSeconds.Set(3600)

	assert.True(t, true)
}
