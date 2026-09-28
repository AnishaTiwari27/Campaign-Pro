package platform

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracer wires up OpenTelemetry tracing for a service and returns a
// shutdown func to flush/close on exit.
//
// Exporter selection is env-var-gated on purpose: with OTEL_EXPORTER_OTLP_ENDPOINT
// unset (the local-dev default), spans print to stdout — real spans, zero
// extra infra, nothing to install. Set that env var (plus OTEL_EXPORTER_OTLP_HEADERS
// for a Dynatrace API token, e.g. "Authorization=Api-Token <token>") and the
// exact same code ships spans to a real OTLP collector/Dynatrace tenant
// instead — flipping this on later is a config change, not a rewrite.
func InitTracer(serviceName string) (func(context.Context) error, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, err
	}

	var exporter sdktrace.SpanExporter
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		exporter, err = otlptracehttp.New(ctx) // reads OTEL_EXPORTER_OTLP_* env vars itself
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
