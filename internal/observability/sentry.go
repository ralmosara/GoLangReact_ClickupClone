package observability

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

// SentryConfig is env-driven. If DSN is empty, InitSentry is a no-op and the
// returned middleware passes requests through unchanged. This keeps
// development setups (where most engineers don't want their stack traces
// shipped to a SaaS) zero-config.
type SentryConfig struct {
	DSN         string
	Environment string
	Release     string
	SampleRate  float64 // 0..1 — share of events to actually transmit
}

// InitSentry initializes the global Sentry client and returns:
//   - a flush function suitable for main.go's shutdown sequence
//   - an HTTP middleware that recovers panics, attaches request context, and
//     reports unhandled errors written via Hub-from-context
//
// When DSN is empty, both returns are pass-through no-ops.
func InitSentry(cfg SentryConfig, log *slog.Logger) (flush func(time.Duration) bool, mw func(http.Handler) http.Handler, err error) {
	if cfg.DSN == "" {
		if log != nil {
			log.Info("sentry disabled (SENTRY_DSN unset)")
		}
		return func(time.Duration) bool { return true }, func(next http.Handler) http.Handler { return next }, nil
	}
	rate := cfg.SampleRate
	if rate <= 0 || rate > 1 {
		rate = 1.0
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		SampleRate:       rate,
		AttachStacktrace: true,
		EnableTracing:    false, // tracing handled by OpenTelemetry, not Sentry
	}); err != nil {
		return nil, nil, err
	}
	if log != nil {
		log.Info("sentry enabled", "environment", cfg.Environment, "release", cfg.Release)
	}
	handler := sentryhttp.New(sentryhttp.Options{Repanic: true})
	return sentry.Flush, handler.Handle, nil
}
