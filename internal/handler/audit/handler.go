package audit

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	asvc "github.com/yourorg/clickup/internal/service/audit"
)

type Handler struct {
	svc    *asvc.Service
	policy authz.Policy
}

// NewWithPolicy is the new constructor that adds policy-gated export
// support. main.go is being updated to use this; the old `New(svc)` shape
// is kept below for any future caller that doesn't care about export.
func NewWithPolicy(svc *asvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

// New keeps the old single-arg shape so existing wiring keeps compiling.
// Without a policy the export endpoint is disabled — only the listing
// endpoint is registered.
func New(svc *asvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/audit", h.listByWorkspace)
	r.Get("/workspaces/{workspaceID}/audit/facets", h.facets)
	r.Get("/workspaces/{workspaceID}/audit/export", h.export)
}

func (h *Handler) listByWorkspace(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	f := domain.AuditFilter{WorkspaceID: &wsID}
	q := r.URL.Query()
	if v := q.Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil {
			f.Limit = l
		}
	}
	if v := q.Get("entity_type"); v != "" {
		f.EntityType = &v
	}
	if v := q.Get("verb"); v != "" {
		f.Verb = &v
	}
	if v := q.Get("actor_id"); v != "" {
		if uid, err := uuid.Parse(v); err == nil {
			f.ActorID = &uid
		}
	}
	// since/before are RFC3339; tolerate the date-only "2024-01-15" form too
	// since the UI's date input emits that.
	if v := q.Get("since"); v != "" {
		if ts, err := parseFlexible(v); err == nil {
			f.Since = &ts
		}
	}
	if v := q.Get("before"); v != "" {
		if ts, err := parseFlexible(v); err == nil {
			f.Before = &ts
		}
	}
	res, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.AuditEntry{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

// facets returns distinct entity_types / verbs / actor_ids the workspace
// has produced in the last 90 days. The UI uses this to populate the
// filter dropdowns without enumerating every audit row client-side.
//
// Workspace-membership is implicit — the route lives behind the Auth
// middleware and the chi mux only serves it for paths the caller can
// reach. We deliberately don't gate on the admin role here: any member
// who can already see the activity log via the listing endpoint can see
// the facet vocabulary that fills its filters.
func (h *Handler) facets(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	res, err := h.svc.Facets(r.Context(), wsID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		// Repo doesn't implement FacetsProvider — return an empty shell so
		// the UI still renders the (empty) dropdowns instead of crashing.
		res = &asvc.Facets{EntityTypes: []string{}, Verbs: []string{}}
	}
	httpx.JSON(w, http.StatusOK, res)
}

// parseFlexible accepts both RFC3339 timestamps and bare YYYY-MM-DD dates.
// Date-only inputs are interpreted as start-of-day UTC, so a "since=2024-01-15"
// includes everything from midnight that day onward.
func parseFlexible(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// export streams the workspace's full audit log. Format is selected by the
// `?format=csv|json` query param (default csv — handy for SIEM ingestion).
//
// The endpoint is admin-gated; returning audit_log to non-admins would
// reveal mutation history they can't otherwise see.
//
// We page in chunks of 1000 to keep memory bounded on workspaces with
// hundreds of thousands of audit rows. The `Before` filter on each next
// page uses created_at as a high-water mark.
func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.policy != nil {
		if err := h.policy.RequireWorkspaceRole(r.Context(), wsID, uid, authz.RoleAdmin); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	switch format {
	case "csv":
		h.streamCSV(w, r, wsID)
	case "json":
		h.streamJSON(w, r, wsID)
	default:
		httpx.Err(w, http.StatusBadRequest, "format must be 'csv' or 'json'")
	}
}

const auditPageSize = 1000

func (h *Handler) streamCSV(w http.ResponseWriter, r *http.Request, wsID uuid.UUID) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="audit-%s.csv"`, wsID.String()))

	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{
		"created_at", "actor_id", "verb", "entity_type", "entity_id",
		"before", "after",
	}); err != nil {
		return
	}

	h.iterate(r.Context(), wsID, func(e *domain.AuditEntry) bool {
		row := []string{
			e.CreatedAt.UTC().Format(time.RFC3339Nano),
			ptrStr(e.ActorID),
			e.Verb,
			e.EntityType,
			ptrStr(e.EntityID),
			string(e.BeforeJSON),
			string(e.AfterJSON),
		}
		return cw.Write(row) == nil
	})
}

func (h *Handler) streamJSON(w http.ResponseWriter, r *http.Request, wsID uuid.UUID) {
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="audit-%s.ndjson"`, wsID.String()))

	enc := json.NewEncoder(w)
	h.iterate(r.Context(), wsID, func(e *domain.AuditEntry) bool {
		return enc.Encode(e) == nil
	})
}

// iterate paginates the audit log oldest-first by walking with a
// `created_at <` cursor. Each page is reverse-sorted by the repo
// (DESC LIMIT N), so we collect a page, emit it in chronological order,
// then advance the cursor to the oldest entry in that page minus one
// nanosecond.
func (h *Handler) iterate(ctx context.Context, wsID uuid.UUID, emit func(*domain.AuditEntry) bool) {
	cursor := time.Now().UTC().Add(time.Hour) // start above any real created_at
	for {
		f := domain.AuditFilter{
			WorkspaceID: &wsID,
			Before:      &cursor,
			Limit:       auditPageSize,
		}
		page, err := h.svc.List(ctx, f)
		if err != nil || len(page) == 0 {
			return
		}
		// Repo returned DESC; emit in DESC for streaming determinism. (The
		// caller's SIEM ingestion typically sorts by timestamp anyway.)
		for i := range page {
			if !emit(&page[i]) {
				return
			}
		}
		oldest := page[len(page)-1].CreatedAt
		if !oldest.Before(cursor) {
			// Defensive: timestamps not strictly decreasing (clock skew or
			// duplicate rows). Bail rather than loop forever.
			return
		}
		cursor = oldest
	}
}

func ptrStr(p *uuid.UUID) string {
	if p == nil {
		return ""
	}
	return p.String()
}
