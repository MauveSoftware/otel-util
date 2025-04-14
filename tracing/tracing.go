package tracing

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var tracer trace.Tracer

// Tracer returns the configured tracer for the application
func Tracer() trace.Tracer {
	return tracer
}

// Init initializes tracing with the given application name, version, and collector endpoint.
// When enabled is false, it sets up a no-operation tracer.
// It returns a cleanup function for proper shutdown and an error if initialization fails.
func Init(ctx context.Context, name, ver string, enabled bool, collectorEndpoint string) (func(), error) {
	tracer = otel.GetTracerProvider().Tracer(
		name,
		trace.WithSchemaURL(semconv.SchemaURL),
	)

	if !enabled {
		return initTracingWithNoop()
	}

	return initTracingToCollector(ctx, name, ver, collectorEndpoint)
}

func initTracingWithNoop() (func(), error) {
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)

	return func() {}, nil
}

func initTracingToCollector(ctx context.Context, name, ver, collectorEndpoint string) (func(), error) {
	logrus.Infof("Initialize tracing (endpoint: %s)", collectorEndpoint)

	cl := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(collectorEndpoint),
	)
	exp, err := otlptrace.New(ctx, cl)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC tracing exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exp)
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(name),
		semconv.ServiceVersionKey.String(ver),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return shutdownTraceProvider(ctx, tp.Shutdown), nil
}

func shutdownTraceProvider(ctx context.Context, shutdownFunc func(ctx context.Context) error) func() {
	return func() {
		if err := shutdownFunc(ctx); err != nil {
			logrus.Errorf("failed to shutdown TracerProvider: %v", err)
		}
	}
}
