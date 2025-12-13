package observability

import (
	"context"
	"io"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TracingConfig defines the settings used to initialize OpenTelemetry tracing.
type TracingConfig struct {
	ServiceName string
	Writer      io.Writer
}

// SetupTracing configures a global tracer provider and returns a shutdown
// function that should be called when the application exits.
func SetupTracing(ctx context.Context, cfg TracingConfig) (func(context.Context) error, error) {
	writer := cfg.Writer
	if writer == nil {
		writer = os.Stdout
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(writer),
	)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return tracerProvider.Shutdown, nil
}
