package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// InitTracer configures a secure, high-performance OTLP/HTTP distributed trace exporter.
// Time Complexity: O(1)
// Space Complexity: O(1)
func InitTracer(ctx context.Context, serviceName, collectorURL string, logger *slog.Logger) (*sdktrace.TracerProvider, error) {
	logger.InfoContext(ctx, "initializing OpenTelemetry distributed tracer over HTTP",
		slog.String("service_name", serviceName),
		slog.String("collector_url", collectorURL),
	)

	if collectorURL == "" {
		collectorURL = "localhost:4318" // Standard OTel collector HTTP port is 4318 (gRPC is 4317)
	}

	// 1. Configure OTLP/HTTP Trace Exporter (licensing-safe and audit-friendly)
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(collectorURL),
		otlptracehttp.WithInsecure(), // Default for local; TLS is handled at mesh ingress/sidecar levels
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP HTTP trace exporter: %w", err)
	}

	// 2. Define service resources
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTel resource attributes: %w", err)
	}

	// 3. Instantiate TracerProvider with Batch Span Processor
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 100% sampling for development and auditing compliance
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// 4. Set global trace registries
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.InfoContext(ctx, "OpenTelemetry distributed tracer initialized successfully")
	return tp, nil
}
