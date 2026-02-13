package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestInitTracing(t *testing.T) {
	cleanup, err := InitTracing("test-service", "1.0.0")

	assert.NoError(t, err, "InitTracing should not return error")
	assert.NotNil(t, cleanup, "cleanup function should not be nil")

	// Call cleanup
	if cleanup != nil {
		cleanup()
	}
}

func TestGetTracer(t *testing.T) {
	// Initialize tracing first
	cleanup, err := InitTracing("test-service", "1.0.0")
	assert.NoError(t, err)
	defer cleanup()

	tracer := GetTracer()
	assert.NotNil(t, tracer, "GetTracer should return a tracer")
}

func TestGetTracerWithoutInit(t *testing.T) {
	// GetTracer should return a default tracer even if not initialized
	tracer := GetTracer()
	assert.NotNil(t, tracer, "GetTracer should return a tracer even without init")
}

func TestStartSpan(t *testing.T) {
	// Initialize tracing
	cleanup, err := InitTracing("test-service", "1.0.0")
	assert.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	// Start a span
	newCtx, span := StartSpan(ctx, "test-operation")

	assert.NotNil(t, newCtx, "context should not be nil")
	assert.NotNil(t, span, "span should not be nil")
	assert.NotEqual(t, ctx, newCtx, "new context should be different from original")

	// End the span
	span.End()
}

func TestStartSpanCreatesValidSpan(t *testing.T) {
	cleanup, err := InitTracing("test-service", "1.0.0")
	assert.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test-operation")

	// Verify span is valid
	assert.True(t, span.SpanContext().IsValid(), "span context should be valid")
	assert.True(t, span.IsRecording(), "span should be recording")

	// Verify span ID exists
	assert.NotEqual(t, trace.SpanID{}, span.SpanContext().SpanID(), "span ID should not be empty")

	// Verify trace ID exists
	assert.NotEqual(t, trace.TraceID{}, span.SpanContext().TraceID(), "trace ID should not be empty")

	span.End()

	// After ending, span should still be valid but not recording
	assert.True(t, span.SpanContext().IsValid(), "span context should still be valid after ending")

	// Get span from context
	spanFromCtx := trace.SpanFromContext(newCtx)
	assert.Equal(t, span.SpanContext().SpanID(), spanFromCtx.SpanContext().SpanID(),
		"span from context should match original span")
}

func TestNestedSpans(t *testing.T) {
	cleanup, err := InitTracing("test-service", "1.0.0")
	assert.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	// Start parent span
	ctx1, span1 := StartSpan(ctx, "parent-operation")
	assert.NotNil(t, span1)

	// Start child span
	ctx2, span2 := StartSpan(ctx1, "child-operation")
	assert.NotNil(t, span2)

	// Child should have same trace ID as parent but different span ID
	assert.Equal(t, span1.SpanContext().TraceID(), span2.SpanContext().TraceID(),
		"child span should have same trace ID as parent")
	assert.NotEqual(t, span1.SpanContext().SpanID(), span2.SpanContext().SpanID(),
		"child span should have different span ID from parent")

	// End spans in reverse order (child first, then parent)
	span2.End()
	span1.End()

	_ = ctx2 // Use ctx2 to avoid unused variable warning
}
