package metrics

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

var appName string

// Meter returns the meter of the application from the configured meter provider
func Meter() metric.Meter {
	return otel.GetMeterProvider().Meter(appName, metric.WithSchemaURL(semconv.SchemaURL))
}

// Init initializes metrics with the given application name, version, and collector endpoint.
// When no collector address is set, it sets up a no-operation meter provider.
// It returns a cleanup function for proper shutdown and an error if initialization fails.
func Init(ctx context.Context, name, version, collectorEndpoint string, attrs ...attribute.KeyValue) (func(), error) {
	appName = name

	if len(collectorEndpoint) == 0 {
		otel.SetMeterProvider(noop.NewMeterProvider())
		return func() {}, nil
	}

	return initMetricsToCollector(ctx, name, version, collectorEndpoint, attrs...)
}

func initMetricsToCollector(ctx context.Context, name, version, collectorEndpoint string, attrs ...attribute.KeyValue) (func(), error) {
	logrus.Infof("Initialize metrics (endpoint: %s)", collectorEndpoint)

	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(collectorEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC metric exporter: %w", err)
	}

	attrs = append(attrs,
		semconv.ServiceName(name),
		semconv.ServiceVersion(version),
	)
	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
	)
	otel.SetMeterProvider(mp)

	return shutdownMeterProvider(ctx, mp.Shutdown), nil
}

func shutdownMeterProvider(ctx context.Context, shutdownFunc func(ctx context.Context) error) func() {
	return func() {
		if err := shutdownFunc(ctx); err != nil {
			logrus.Errorf("failed to shutdown MeterProvider: %v", err)
		}
	}
}
