// Package observability holds cross-cutting infrastructure: structured-log
// context propagation, Prometheus metrics, OpenTelemetry tracing, and Sentry.
//
// None of these have a hard runtime dependency: every initializer falls back
// to a no-op if its env-driven config is unset, so a fresh checkout still
// boots without an OTLP collector or Sentry DSN.
package observability

import (
	"context"
	"log/slog"
)

type loggerCtxKey struct{}

// WithLogger derives a context that carries the given logger. Subsequent
// FromContext(ctx) calls return this logger; if no logger is attached they
// fall back to slog.Default(). The middleware layer attaches a request-scoped
// logger (with request_id, method, path) at the start of every HTTP request
// so any downstream layer that uses observability.FromContext gets enriched
// log lines for free.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerCtxKey{}, l)
}

// FromContext returns the request-scoped logger attached to ctx, or
// slog.Default() if none. Always safe to call.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if v, ok := ctx.Value(loggerCtxKey{}).(*slog.Logger); ok && v != nil {
		return v
	}
	return slog.Default()
}
