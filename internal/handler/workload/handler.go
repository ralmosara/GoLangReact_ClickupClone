// Package workload exposes the workload/capacity report.
//
//   GET /workspaces/{workspaceID}/workload?from=...&to=...
//
// Both bounds are RFC3339; both default — to=now, from=now-14d — so a
// bare GET still returns useful data. The window applies to due_at on
// open tasks and completed_at on closed ones.
package workload

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/httpx"
	wsvc "github.com/yourorg/clickup/internal/service/workload"
)

type Handler struct{ svc *wsvc.Service }

func New(svc *wsvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/workload", h.report)
}

func (h *Handler) report(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	q := r.URL.Query()
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -14)
	to := now.AddDate(0, 0, 14) // forward window — capture upcoming due dates too

	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		}
	}

	res, err := h.svc.Report(r.Context(), wsvc.Input{WorkspaceID: wsID, From: from, To: to})
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
