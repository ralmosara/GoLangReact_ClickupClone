package middleware

import (
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/yourorg/clickup/internal/observability"
)

// RequestLogger reads the request ID emitted by chimw.RequestID, derives a
// per-request *slog.Logger that carries request_id/method/path, and stores
// it on the request context. Anything downstream that reaches for
// observability.FromContext(ctx) gets enriched log lines.
//
// This sits *after* chimw.RequestID in the middleware chain (which is where
// main.go installs it).
func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	if base == nil {
		base = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := chimw.GetReqID(r.Context())
			l := base.With(
				slog.String("request_id", rid),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)
			ctx := observability.WithLogger(r.Context(), l)
			// Surface the request id to clients so support engineers can
			// correlate a user-reported error to a backend log line.
			if rid != "" {
				w.Header().Set("X-Request-ID", rid)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
