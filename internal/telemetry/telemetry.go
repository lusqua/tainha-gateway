package telemetry

import (
	"context"
	"log/slog"
	"time"

	"github.com/aguiar-sh/tainha/internal/config"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type Shutdown func(ctx context.Context) error

// Setup initializes OpenTelemetry tracing and metrics.
// Returns a shutdown function to flush and close exporters.
func Setup(ctx context.Context, cfg config.TelemetryConfig) (Shutdown, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	var shutdowns []func(ctx context.Context) error

	// Traces
	if cfg.ExporterEndpoint != "" {
		traceExporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.ExporterEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter, sdktrace.WithBatchTimeout(5*time.Second)),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, tp.Shutdown)

		slog.Info("otel traces enabled", "endpoint", cfg.ExporterEndpoint)
	}

	// W3C trace context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Metrics (Prometheus)
	promExp, err := promexporter.New()
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExp),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	shutdowns = append(shutdowns, mp.Shutdown)

	slog.Info("otel metrics enabled (prometheus /metrics)")

	shutdown := func(ctx context.Context) error {
		for _, fn := range shutdowns {
			if err := fn(ctx); err != nil {
				slog.Error("otel shutdown error", "error", err)
			}
		}
		return nil
	}

	return shutdown, nil
}
