package middleware

import (
	"bufio"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/yourorg/clickup/internal/observability"
)

// statusRecorder wraps ResponseWriter to capture the status code emitted by
// the handler. It also forwards optional interfaces (Hijacker, Flusher) so
// middlewares later in the chain — notably the websocket upgrade, which needs
// to hijack the underlying TCP connection — keep working.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// Hijack is required by gorilla/websocket to take over the connection for the
// WebSocket upgrade. Without this, the embedded ResponseWriter's Hijacker
// interface is hidden by this wrapper and the upgrade fails with
// "websocket: response does not implement http.Hijacker".
func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := s.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("underlying ResponseWriter is not a Hijacker")
}

// Flush is needed if any endpoint streams (SSE, chunked responses).
func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Logger emits one structured log line per request. It pulls the per-request
// logger that RequestLogger attached to the context (which carries
// request_id/method/path) and falls back to the supplied base logger if no
// per-request logger is present — keeping it backwards-compatible with code
// paths that don't go through chi (e.g. raw mux routes).
func Logger(base *slog.Logger) func(http.Handler) http.Handler {
	if base == nil {
		base = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			l := observability.FromContext(r.Context())
			if l == slog.Default() {
				l = base
			}
			l.Info("http",
				"status", rec.status,
				"bytes", rec.bytes,
				"dur_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
