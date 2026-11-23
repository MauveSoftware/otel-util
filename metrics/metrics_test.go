package metrics

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestInitWithoutCollectorUsesNoop(t *testing.T) {
	shutdown, err := Init(context.Background(), "test-app", "1.0.0", "")
	if err != nil {
		t.Fatalf("Init without collector failed: %v", err)
	}
	defer shutdown()

	if _, ok := otel.GetMeterProvider().(noop.MeterProvider); !ok {
		t.Errorf("want a no-op meter provider without collector, got %T", otel.GetMeterProvider())
	}
	if _, err := Meter().Int64Counter("test.counter"); err != nil {
		t.Errorf("Meter must be usable with the no-op provider: %v", err)
	}
}

func TestInitWithCollectorSetsSDKProvider(t *testing.T) {
	// The gRPC connection is established lazily, so an unreachable endpoint is fine here. The
	// short timeout bounds the final flush on shutdown, which can't reach the collector.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	shutdown, err := Init(ctx, "test-app", "1.0.0", "127.0.0.1:1")
	if err != nil {
		t.Fatalf("Init with collector failed: %v", err)
	}

	if _, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider); !ok {
		t.Errorf("want an SDK meter provider with collector, got %T", otel.GetMeterProvider())
	}
	if _, err := Meter().Int64Counter("test.counter"); err != nil {
		t.Errorf("Meter must be usable with the SDK provider: %v", err)
	}
	shutdown()
}
