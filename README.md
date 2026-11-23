# otel-util
Library to reduce boilerplate code when using OTEL

Packages (each with an `Init(ctx, name, version, collectorEndpoint, attrs...)` that falls back
to a no-op provider without collector endpoint):

- `tracing`: traces via OTLP/gRPC, `tracing.Tracer()`
- `logging`: logrus logs via OTLP/gRPC, `logging.Logger()`
- `metrics`: metrics via OTLP/gRPC, `metrics.Meter()`
