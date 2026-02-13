package observability

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// TracingMiddleware adds OpenTelemetry tracing to HTTP requests
func TracingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			ctx := req.Context()

			// Start span
			spanName := req.Method + " " + c.Path()
			ctx, span := GetTracer().Start(ctx, spanName)
			defer span.End()

			// Set span attributes
			span.SetAttributes(
				attribute.String("http.method", req.Method),
				attribute.String("http.url", req.URL.String()),
				attribute.String("http.scheme", req.URL.Scheme),
				attribute.String("http.host", req.Host),
				attribute.String("http.path", c.Path()),
				attribute.String("http.user_agent", req.UserAgent()),
				attribute.String("http.remote_addr", c.RealIP()),
			)

			// Add custom headers to span
			if featureName := req.Header.Get("X-Feature-Name"); featureName != "" {
				span.SetAttributes(attribute.String("mocktool.feature", featureName))
			}
			if accountID := req.Header.Get("X-Account-Id"); accountID != "" {
				span.SetAttributes(attribute.String("mocktool.account_id", accountID))
			}

			// Update context in request
			c.SetRequest(req.WithContext(ctx))

			// Execute handler
			err := next(c)

			// Set response attributes
			status := c.Response().Status
			if status == 0 {
				status = 200
			}
			span.SetAttributes(
				attribute.Int("http.status_code", status),
				attribute.Int64("http.response_size", c.Response().Size),
			)

			// Record error if present
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())

				if he, ok := err.(*echo.HTTPError); ok {
					span.SetAttributes(
						attribute.Int("http.status_code", he.Code),
						attribute.String("http.error_message", he.Error()),
					)
				}
			} else if status >= 400 {
				span.SetStatus(codes.Error, "HTTP error status")
			} else {
				span.SetStatus(codes.Ok, "")
			}

			return err
		}
	}
}

// TraceSpan is a helper to easily add tracing to any function
func TraceSpan(ctx echo.Context, spanName string, fn func() error) error {
	reqCtx := ctx.Request().Context()
	newCtx, span := GetTracer().Start(reqCtx, spanName)
	defer span.End()

	// Update request context
	ctx.SetRequest(ctx.Request().WithContext(newCtx))

	err := fn()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}
