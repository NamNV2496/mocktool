package middleware

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/namnv2496/mocktool/internal/repository/health"
	"github.com/namnv2496/mocktool/pkg/observability"
)

type LoadShedding struct {
	maxCurrency    int64
	warningLatency int64
	maxLatency     int64
	current        int64
}

func NewLoadShedding(
	maxCurrency int64,
	warningLatency int64,
	maxLatency int64,
) *LoadShedding {
	return &LoadShedding{
		maxCurrency:    maxCurrency,
		warningLatency: warningLatency,
		maxLatency:     maxLatency,
	}
}

func (l *LoadShedding) IsOverload() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Concurrency-based load shedding
			if atomic.LoadInt64(&l.current) >= l.maxCurrency {
				observability.LoadSheddingCount.Inc()
				return c.JSON(http.StatusServiceUnavailable, map[string]string{
					"error": "server overloaded",
				})
			}
			atomic.AddInt64(&l.current, 1)
			defer atomic.AddInt64(&l.current, -1)
			// Latency-based adaptive shedding
			status, latency := health.IsOverloaded(l.warningLatency, l.maxLatency)
			switch status {
			case 1:
				observability.LoadSheddingDelayLatencyCount.Inc()
				delay := calculateDelay(latency)
				time.Sleep(delay)
			case 2:
				observability.LoadSheddingLatencyCount.Inc()
				return c.JSON(http.StatusServiceUnavailable, map[string]string{
					"error": "system overloaded - load shedding active",
				})
			}
			return next(c)
		}
	}
}

func calculateDelay(currentLatency int64) time.Duration {
	var delayThreadhold int64 = 1000
	var maxDelayThreadhold int64 = 3000
	if currentLatency == 0 {
		return time.Duration(delayThreadhold) * time.Millisecond
	}
	loadRatio := float64(currentLatency-delayThreadhold) / float64(maxDelayThreadhold)
	// 0% overload → 0ms
	// 100% overload → 100ms
	maxDelay := 100 * time.Millisecond
	delay := time.Duration(loadRatio * float64(maxDelay))
	if delay < 0 {
		return 0
	}
	return delay
}
