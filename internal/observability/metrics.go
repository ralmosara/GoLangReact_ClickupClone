package observability

import (
	"context"
	"net/http"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics is the Prometheus registry plus the small set of app-level series
// the rest of the codebase pushes into. Keeping the registry on a struct
// (rather than promauto + the global default registry) makes tests trivial:
// each test gets a fresh Metrics with its own registry.
type Metrics struct {
	Registry *prometheus.Registry

	// HTTP RED — rate / errors / duration histogram per route+method+status.
	// Cardinality is bounded because chi templated routes ("/tasks/{id}") are
	// recorded as their template, not the rendered URL.
	HTTPRequests *prometheus.CounterVec
	HTTPDuration *prometheus.HistogramVec

	// WebSocket hub — fanout health.
	WSConnections prometheus.Gauge
	WSRooms       prometheus.Gauge
	WSPublished   *prometheus.CounterVec // by room type
	WSDropped     *prometheus.CounterVec // by reason

	// Automation engine — queue depth + per-trigger fire counter.
	AutoQueueDepth prometheus.Gauge
	AutoFires      *prometheus.CounterVec // by trigger
	AutoErrors     *prometheus.CounterVec // by action

	// Auth surface — login/MFA/SSO.
	AuthAttempts *prometheus.CounterVec // by event,outcome
}

// NewMetrics builds a fresh registry and registers app + go-runtime + process
// collectors. Pass the resulting *Metrics through DI so middleware/services
// can record. Pgx pool stats are wired separately via WirePgxPoolStats so the
// registry doesn't need to import pgx.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	// Default Go-runtime + process metrics — these are the table stakes any
	// SRE expects on /metrics.
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	m := &Metrics{
		Registry: reg,
		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests processed, partitioned by status, method, and route template.",
		}, []string{"method", "route", "status"}),
		HTTPDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds.",
			Buckets: prometheus.ExponentialBuckets(0.005, 2, 12), // 5ms .. ~10s
		}, []string{"method", "route"}),

		WSConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ws_connections",
			Help: "Number of currently-connected WebSocket clients.",
		}),
		WSRooms: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ws_rooms",
			Help: "Number of currently-active WebSocket rooms.",
		}),
		WSPublished: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ws_events_published_total",
			Help: "Events fanned out by the WebSocket hub.",
		}, []string{"room_kind"}),
		WSDropped: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ws_events_dropped_total",
			Help: "Events dropped by the WebSocket hub.",
		}, []string{"reason"}),

		AutoQueueDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "automation_queue_depth",
			Help: "Number of events queued in the automation engine.",
		}),
		AutoFires: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "automation_fires_total",
			Help: "Number of automation runs by trigger.",
		}, []string{"trigger"}),
		AutoErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "automation_errors_total",
			Help: "Number of failed automation actions by action type.",
		}, []string{"action"}),

		AuthAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "auth_attempts_total",
			Help: "Authentication events partitioned by event (login,mfa_verify,oidc_callback) and outcome (ok,denied,error).",
		}, []string{"event", "outcome"}),
	}

	reg.MustRegister(
		m.HTTPRequests, m.HTTPDuration,
		m.WSConnections, m.WSRooms, m.WSPublished, m.WSDropped,
		m.AutoQueueDepth, m.AutoFires, m.AutoErrors,
		m.AuthAttempts,
	)
	return m
}

// Handler returns the http.Handler for /metrics. Fronts the registry with
// promhttp; safe to mount on the same chi router as the rest of the API.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// WirePgxPoolStats registers a one-shot pool-stats collector. Each scrape
// pulls live counters off the pgxpool.Pool — no background goroutine, no
// per-query overhead.
func (m *Metrics) WirePgxPoolStats(pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	m.Registry.MustRegister(&pgxPoolCollector{pool: pool})
}

type pgxPoolCollector struct {
	once sync.Once
	pool *pgxpool.Pool

	acquired   *prometheus.Desc
	idle       *prometheus.Desc
	total      *prometheus.Desc
	maxConns   *prometheus.Desc
	acquireDur *prometheus.Desc
	canceled   *prometheus.Desc
}

func (c *pgxPoolCollector) describe() {
	c.acquired = prometheus.NewDesc("pgx_pool_acquired_conns", "Currently-acquired connections.", nil, nil)
	c.idle = prometheus.NewDesc("pgx_pool_idle_conns", "Currently-idle connections.", nil, nil)
	c.total = prometheus.NewDesc("pgx_pool_total_conns", "Total connections in the pool.", nil, nil)
	c.maxConns = prometheus.NewDesc("pgx_pool_max_conns", "Configured maximum.", nil, nil)
	c.acquireDur = prometheus.NewDesc("pgx_pool_acquire_duration_seconds_total", "Cumulative time waiting for a connection.", nil, nil)
	c.canceled = prometheus.NewDesc("pgx_pool_canceled_acquires_total", "Cumulative canceled acquires.", nil, nil)
}

func (c *pgxPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	c.once.Do(c.describe)
	ch <- c.acquired
	ch <- c.idle
	ch <- c.total
	ch <- c.maxConns
	ch <- c.acquireDur
	ch <- c.canceled
}

func (c *pgxPoolCollector) Collect(ch chan<- prometheus.Metric) {
	c.once.Do(c.describe)
	if c.pool == nil {
		return
	}
	s := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.acquired, prometheus.GaugeValue, float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idle, prometheus.GaugeValue, float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.total, prometheus.GaugeValue, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.acquireDur, prometheus.CounterValue, s.AcquireDuration().Seconds())
	ch <- prometheus.MustNewConstMetric(c.canceled, prometheus.CounterValue, float64(s.CanceledAcquireCount()))
}

// RecordAuthAttempt is a small helper for handlers that want to push to the
// auth counter without taking a hard dependency on the prometheus client.
// Nil-safe so test scaffolds can leave Metrics unset.
func (m *Metrics) RecordAuthAttempt(event, outcome string) {
	if m == nil {
		return
	}
	m.AuthAttempts.WithLabelValues(event, outcome).Inc()
}

// Suppress the unused-context linter — kept on the signature to match the
// shape of future per-collect background pulls.
var _ = func(ctx context.Context) {}
