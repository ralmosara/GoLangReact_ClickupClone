package observability

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// TraceConfig is the env-driven configuration for OpenTelemetry. If
// OTLPEndpoint is empty, InitTracing returns a no-op shutdown function and
// otel.Tracer keeps using its default no-op provider — i.e. zero overhead.
type TraceConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string // host:port, e.g. "otel-collector:4318"
	OTLPInsecure   bool
	SampleRatio    float64 // 0..1; 0 disables, 1 records every span
}

// InitTracing wires the global OpenTelemetry tracer provider and the
// W3C Trace-Context + Baggage propagators. The returned shutdown function
// flushes the exporter on graceful shutdown — main.go should defer it.
//
// Behaviour:
//   - OTLPEndpoint == "": no-op provider. Returns nil shutdown, no error.
//   - OTLPEndpoint set:   OTLP/HTTP exporter, parent-based sampler with the
//     given ratio, registered as the global provider.
func InitTracing(ctx context.Context, cfg TraceConfig, log *slog.Logger) (func(context.Context) error, error) {
	if cfg.OTLPEndpoint == "" {
		if log != nil {
			log.Info("otel tracing disabled (OTEL_EXPORTER_OTLP_ENDPOINT unset)")
		}
		// Still install the propagators so any incoming traceparent header is
		// honoured even if we don't export. Cheap and forwards-compatible.
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		))
		return func(context.Context) error { return nil }, nil
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.OTLPEndpoint)}
	if cfg.OTLPInsecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
			semconv.HostName(hostname()),
		),
	)
	if err != nil {
		return nil, err
	}

	ratio := cfg.SampleRatio
	if ratio <= 0 {
		ratio = 0.1
	}
	if ratio > 1 {
		ratio = 1
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	if log != nil {
		log.Info("otel tracing enabled",
			"endpoint", cfg.OTLPEndpoint,
			"sample_ratio", ratio,
			"insecure", cfg.OTLPInsecure,
		)
	}
	return tp.Shutdown, nil
}

func hostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}
