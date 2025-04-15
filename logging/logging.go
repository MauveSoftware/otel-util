package logging

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/noop"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

var appName string

// Logger returns the configured log.LoggerProvider instance
func Logger() log.Logger {
	return global.Logger(appName)
}

// Init initializes the logger with the given application name, version, and collector endpoint.
// When no collector address is set, it sets up a no-operation logger.
// It returns a cleanup function for proper shutdown and an error if initialization fails.
func Init(ctx context.Context, name, version, collectorEndpoint string, attrs ...attribute.KeyValue) (func(), error) {
	appName = name

	if len(collectorEndpoint) == 0 {
		setLoggerProvider(noop.NewLoggerProvider())
		return func() {}, nil
	}

	return initLogger(ctx, name, version, collectorEndpoint, attrs...)
}

func setLoggerProvider(lp log.LoggerProvider) {
	global.SetLoggerProvider(lp)

	hook := otellogrus.NewHook(appName, otellogrus.WithLoggerProvider(lp))
	logrus.AddHook(hook)
}

func initLogger(ctx context.Context, name, version, collectorEndpoint string, attrs ...attribute.KeyValue) (func(), error) {
	exp, err := otlploggrpc.New(ctx,
		otlploggrpc.WithInsecure(),
		otlploggrpc.WithEndpoint(collectorEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC log exporter: %w", err)
	}

	bsp := logsdk.NewBatchProcessor(exp)

	attrs = append(attrs,
		semconv.ServiceName(name),
		semconv.ServiceVersion(version),
	)
	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)
	lp := logsdk.NewLoggerProvider(
		logsdk.WithResource(res),
		logsdk.WithProcessor(bsp),
	)
	setLoggerProvider(lp)

	return shutdownLogProvider(ctx, lp.Shutdown), nil
}

func shutdownLogProvider(ctx context.Context, shutdownFunc func(ctx context.Context) error) func() {
	return func() {
		if err := shutdownFunc(ctx); err != nil {
			logrus.Errorf("failed to shutdown LogProvider: %v", err)
		}
	}
}
