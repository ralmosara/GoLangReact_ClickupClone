# Service Level Objectives

This document is the contract between engineering and the rest of the
business about how reliable the API is and how the on-call rotation
responds when it isn't.

## Tiered SLOs

| Surface | Availability (28-day) | Latency (p95) | Error budget / 28 days |
| ------- | --------------------- | ------------- | ---------------------- |
| `/api/v1/auth/*` | 99.95 % | 300 ms | 20 min |
| `/api/v1/*` (authenticated read) | 99.9 % | 300 ms | 40 min |
| `/api/v1/*` (authenticated write) | 99.9 % | 800 ms | 40 min |
| `/ws` (WebSocket) | 99.5 % | 500 ms upgrade | 3 h 21 min |
| Background automations (queue → action) | 99.0 % | 60 s | 6 h 43 min |

Availability is measured as `1 - (5xx + non-2xx-from-our-fault) / total`.
Latency is the histogram bucket reported by `http_request_duration_seconds`.

## Error budget policy

- **0 – 50 % budget burned**: business as usual.
- **50 – 75 % burned**: feature work continues; reliability backlog gets a
  lane on the next sprint.
- **75 – 100 % burned**: feature freeze on the affected surface until the
  budget rebuilds. Reliability work is the only acceptable PR target.
- **Budget exhausted**: incident review with a written postmortem, and the
  next 28 days run with halved deploy frequency on that surface.

## How we measure

Every running instance exposes `/metrics` (Prometheus exposition format).
The recommended scrape interval is 15 s. The dashboards listed below are
expected to exist before any phase 2 work ships.

### Required dashboards

1. **API RED** — `sum(rate(http_requests_total[5m])) by (route)`,
   `sum(rate(http_requests_total{status=~"5.."}[5m])) by (route)`,
   `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, route))`.
2. **Postgres pool** — acquired/idle/total against `pgx_pool_max_conns`,
   acquire-duration rate. Alert when `acquired/max > 0.8` for 5 min.
3. **WebSocket hub** — `ws_connections`, `ws_rooms`,
   `rate(ws_events_published_total[1m])`,
   `rate(ws_events_dropped_total[1m])`.
4. **Automations** — `automation_queue_depth`,
   `rate(automation_fires_total[5m])`,
   `rate(automation_errors_total[5m])`.
5. **Auth** — `rate(auth_attempts_total{outcome="denied"}[5m])` over
   `rate(auth_attempts_total[5m])` per event. Alert when denial rate
   exceeds 25 % for 10 min (credential-stuffing signal).

## Probes

| Endpoint | Method | Used by | Expected response |
| -------- | ------ | ------- | ----------------- |
| `/healthz` | GET | Liveness | `200 ok` if the binary is responsive |
| `/readyz` | GET | Readiness | `200` when DB + migrations are up; `503` otherwise |
| `/metrics` | GET | Prometheus scrape | OpenMetrics text |

The Kubernetes/orchestrator readiness probe **must** point at `/readyz`,
never `/healthz`, so a Postgres outage pulls instances out of the load
balancer instead of returning 5xx to users.

## Paging policy

- A symptom-based alert is paged to the on-call when an SLO has burned
  > 5 % of its 28-day budget in the past hour, OR > 1 % in the past
  5 minutes (multi-window, multi-burn-rate).
- Cause-based alerts (e.g. "pgx pool exhausted") are sent to a non-paging
  channel; they exist for triage, not waking people up.

## Postmortem trigger

Any incident that:

- exhausts an SLO budget for the 28-day window, OR
- causes user-visible errors for > 30 min on any tier-1 surface, OR
- requires a manual restore from backup,

triggers a written postmortem within 5 business days, owned by the
incident commander.
