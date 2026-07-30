package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// WithTrace 为日志追加当前 Span 的 Trace ID 和 Span ID。
func WithTrace(ctx context.Context, logger *zap.Logger) *zap.Logger {
	if logger == nil {
		return zap.NewNop()
	}
	if ctx == nil {
		return logger
	}

	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return logger
	}

	return logger.With(
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
	)
}
