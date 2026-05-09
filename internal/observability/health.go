package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Probe is a single dependency health check. The bool tells liveness/readiness;
// the err is included in the JSON body when not nil.
type Probe func(ctx context.Context) error

// HealthHandler returns an http.Handler that runs each named probe and
// reports the result. probesMustPass = false yields a livez-style check
// (always 200 unless the binary itself is broken); true yields a readyz-style
// check (503 if any dep is down).
func HealthHandler(probes map[string]Probe, probesMustPass bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		results := make(map[string]string, len(probes))
		ok := true
		for name, p := range probes {
			if err := p(ctx); err != nil {
				results[name] = "down: " + err.Error()
				ok = false
			} else {
				results[name] = "up"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if probesMustPass && !ok {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     map[bool]string{true: "ok", false: "degraded"}[ok],
			"checks":     results,
			"checked_at": time.Now().UTC().Format(time.RFC3339Nano),
		})
	})
}

// PgxProbe pings the Postgres pool. Failure means the DB is unreachable —
// readiness should fail and the orchestrator should pull traffic.
func PgxProbe(pool *pgxpool.Pool) Probe {
	return func(ctx context.Context) error { return pool.Ping(ctx) }
}

// RedisProbe pings Redis. Failure does not necessarily mean the API is
// unhealthy — rate limiting degrades gracefully — but readyz reports it so
// operators see the dependency state on a single endpoint.
func RedisProbe(rdb *redis.Client) Probe {
	return func(ctx context.Context) error {
		if rdb == nil {
			return nil
		}
		return rdb.Ping(ctx).Err()
	}
}

// MigrationsProbe verifies the schema_migrations table reports at least the
// minimum expected migration version. Pass the highest migration prefix you
// know about (e.g. 30) so a binary deployed against an older DB fails ready
// rather than silently mis-querying.
func MigrationsProbe(pool *pgxpool.Pool, minVersion int) Probe {
	return func(ctx context.Context) error {
		var max int
		err := pool.QueryRow(ctx,
			`SELECT COALESCE(MAX(NULLIF(SUBSTRING(version FROM '^[0-9]+'), '')::int), 0) FROM schema_migrations`,
		).Scan(&max)
		if err != nil {
			return err
		}
		if max < minVersion {
			return errMigrationsBehind{got: max, want: minVersion}
		}
		return nil
	}
}

type errMigrationsBehind struct{ got, want int }

func (e errMigrationsBehind) Error() string {
	return "migrations behind expected version: got " + itoa(e.got) + ", want >=" + itoa(e.want)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [16]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
