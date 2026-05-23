package observability

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer trace.Tracer
)

// InitTracing initializes OpenTelemetry tracing
func InitTracing(serviceName, serviceVersion string, enabled bool) (func(), error) {
	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	var tp *sdktrace.TracerProvider

	if enabled {
		// Create stdout exporter for development
		// In production, replace with Jaeger, Zipkin, or OTLP exporter
		exporter, err := stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return nil, err
		}

		// Create trace provider with exporter
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)
		slog.Info("OpenTelemetry tracing initialized",
			"service", serviceName,
			"version", serviceVersion,
		)
	} else {
		// Create noop trace provider when tracing is disabled
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
		)
		slog.Info("OpenTelemetry tracing disabled")
	}

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Get tracer
	tracer = tp.Tracer("mocktool")

	// Return cleanup function
	cleanup := func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("Error shutting down tracer provider", "error", err)
		}
	}

	return cleanup, nil
}

// GetTracer returns the global tracer
func GetTracer() trace.Tracer {
	if tracer == nil {
		// Fallback to default tracer if not initialized
		return otel.Tracer("mocktool")
	}
	return tracer
}

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, name)
}
