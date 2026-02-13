package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		handler        echo.HandlerFunc
		expectedStatus int
		shouldError    bool
	}{
		{
			name:   "successful GET request",
			method: http.MethodGet,
			path:   "/api/v1/test",
			body:   "",
			handler: func(c echo.Context) error {
				return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
			},
			expectedStatus: http.StatusOK,
			shouldError:    false,
		},
		{
			name:   "successful POST request with body",
			method: http.MethodPost,
			path:   "/api/v1/create",
			body:   `{"name":"test"}`,
			handler: func(c echo.Context) error {
				return c.JSON(http.StatusCreated, map[string]string{"id": "123"})
			},
			expectedStatus: http.StatusCreated,
			shouldError:    false,
		},
		{
			name:   "request with 404 error",
			method: http.MethodGet,
			path:   "/api/v1/notfound",
			body:   "",
			handler: func(c echo.Context) error {
				return echo.NewHTTPError(http.StatusNotFound, "not found")
			},
			expectedStatus: http.StatusNotFound,
			shouldError:    true,
		},
		{
			name:   "request with 500 error",
			method: http.MethodPost,
			path:   "/api/v1/error",
			body:   `{"data":"test"}`,
			handler: func(c echo.Context) error {
				return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
			},
			expectedStatus: http.StatusInternalServerError,
			shouldError:    true,
		},
		{
			name:   "request with 400 bad request",
			method: http.MethodPost,
			path:   "/api/v1/validate",
			body:   "invalid json",
			handler: func(c echo.Context) error {
				return echo.NewHTTPError(http.StatusBadRequest, "bad request")
			},
			expectedStatus: http.StatusBadRequest,
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			// Apply metrics middleware
			e.Use(MetricsMiddleware())

			// Setup route
			e.Add(tt.method, tt.path, tt.handler)

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			// Execute request
			e.ServeHTTP(rec, req)

			// Verify status code (middleware should not change it)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			// Metrics should be recorded (we can't easily verify Prometheus metrics in tests,
			// but we can verify the middleware doesn't break the request flow)
		})
	}
}

func TestMetricsMiddlewarePreservesContext(t *testing.T) {
	e := echo.New()

	// Apply metrics middleware
	e.Use(MetricsMiddleware())

	// Handler that checks context
	e.GET("/test", func(c echo.Context) error {
		assert.NotNil(t, c.Request().Context())
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMetricsMiddlewareHandlesNilError(t *testing.T) {
	e := echo.New()

	e.Use(MetricsMiddleware())

	e.GET("/test", func(c echo.Context) error {
		return nil // No error
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should default to 200 OK when no error and no explicit status
	assert.Equal(t, http.StatusOK, rec.Code)
}
