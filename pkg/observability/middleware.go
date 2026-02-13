package observability

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// MetricsMiddleware collects HTTP metrics for all requests
func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Get request details
			method := c.Request().Method
			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}

			// Measure request size
			if c.Request().ContentLength > 0 {
				HTTPRequestSize.WithLabelValues(method, path).Observe(float64(c.Request().ContentLength))
			}

			// Execute the handler
			err := next(c)

			// Calculate duration
			duration := time.Since(start).Seconds()

			// Determine status code
			status := c.Response().Status
			if status == 0 {
				status = 200
			}
			if err != nil {
				// If there's an error, try to extract status from echo.HTTPError
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				} else {
					status = 500
				}
			}

			// Record metrics
			statusStr := strconv.Itoa(status)
			HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
			HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
			HTTPResponseSize.WithLabelValues(method, path).Observe(float64(c.Response().Size))

			// Record errors
			if err != nil {
				errorType := "internal_error"
				if he, ok := err.(*echo.HTTPError); ok {
					switch he.Code {
					case 400:
						errorType = "bad_request"
					case 401:
						errorType = "unauthorized"
					case 403:
						errorType = "forbidden"
					case 404:
						errorType = "not_found"
					case 429:
						errorType = "rate_limit"
					case 500:
						errorType = "internal_error"
					case 503:
						errorType = "service_unavailable"
					}
				}
				HTTPErrorsTotal.WithLabelValues(method, path, errorType).Inc()
			}

			return err
		}
	}
}
