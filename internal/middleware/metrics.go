package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yourorg/clickup/internal/observability"
)

// HTTPMetrics records the RED triplet (rate / errors / duration) for every
// HTTP request, labelled by chi route template, method, and response status.
// Using the route template (e.g. "/tasks/{id}") rather than the rendered URL
// keeps cardinality bounded.
func HTTPMetrics(m *observability.Metrics) func(http.Handler) http.Handler {
	if m == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			route := "unmatched"
			if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
				route = rctx.RoutePattern()
			}
			status := strconv.Itoa(rec.status)
			m.HTTPRequests.WithLabelValues(r.Method, route, status).Inc()
			m.HTTPDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		})
	}
}
