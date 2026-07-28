package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	otlpEndpoint = "127.0.0.1:14317"
	serviceName  = "gva"
	serviceVer   = "2.9.1"
	initTimeout  = 5 * time.Second
)

// InitTracer 初始化 GVA → Alloy → Tempo 的 Trace 上报链路。
func InitTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporterCtx, cancel := context.WithTimeout(ctx, initTimeout)
	defer cancel()

	exporter, err := otlptracegrpc.New(
		exporterCtx,
		otlptracegrpc.WithEndpoint(otlpEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("创建 OTLP/gRPC Trace Exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVer),
		),
	)
	if err != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), initTimeout)
		defer shutdownCancel()
		_ = exporter.Shutdown(shutdownCtx)
		return nil, fmt.Errorf("创建 OpenTelemetry Resource: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tracerProvider, nil
}
