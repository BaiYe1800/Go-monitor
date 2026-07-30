package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithTrace(t *testing.T) {
	traceID := trace.TraceID{1, 2, 3}
	spanID := trace.SpanID{4, 5, 6}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))
	core, logs := observer.New(zap.InfoLevel)

	WithTrace(ctx, zap.New(core)).Info("request completed")

	fields := logs.All()[0].ContextMap()
	if fields["trace_id"] != traceID.String() {
		t.Fatalf("trace_id = %v, want %s", fields["trace_id"], traceID.String())
	}
	if fields["span_id"] != spanID.String() {
		t.Fatalf("span_id = %v, want %s", fields["span_id"], spanID.String())
	}
}

func TestWithTraceWithoutValidSpanContext(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)

	WithTrace(context.Background(), zap.New(core)).Info("request completed")

	fields := logs.All()[0].ContextMap()
	if _, ok := fields["trace_id"]; ok {
		t.Fatal("invalid SpanContext should not add trace_id")
	}
	if _, ok := fields["span_id"]; ok {
		t.Fatal("invalid SpanContext should not add span_id")
	}
}
