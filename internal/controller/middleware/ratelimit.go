package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/namnv2496/mocktool/internal/repository/ratelimiter"
)

func RateLimitMiddleware(limiter ratelimiter.ILimiter, limitOption string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			switch limitOption {
			case "account_id":
				accountID := c.Request().Header.Get("X-Account-ID")
				if accountID != "" {
					accountKey := "rate_limit:account:" + accountID
					allowed, err := limiter.Allow(ctx, accountKey)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
					}
					if !allowed {
						return echo.NewHTTPError(http.StatusTooManyRequests, "account rate limit exceeded")
					}
				}
			case "ip":
				ip := c.RealIP()
				ipKey := "rate_limit:ip:" + ip

				allowed, err := limiter.Allow(ctx, ipKey)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				if !allowed {
					return echo.NewHTTPError(http.StatusTooManyRequests, "ip rate limit exceeded")
				}
			}

			return next(c)
		}
	}
}
