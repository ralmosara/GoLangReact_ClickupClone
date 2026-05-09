// Package gdpr serves the workspace data-export endpoint. The export is
// gated by RequireWorkspaceRole(owner) — only the workspace owner can pull
// every member's data — and is delivered as application/json with a
// Content-Disposition: attachment header so browsers save it as a file.
package gdpr

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	gdprsvc "github.com/yourorg/clickup/internal/service/gdpr"
)

type Handler struct {
	svc    *gdprsvc.Service
	policy authz.Policy
}

func New(svc *gdprsvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/gdpr/export", h.export)
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	if err := h.policy.RequireWorkspaceRole(r.Context(), wsID, uid, authz.RoleOwner); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	tables, err := h.svc.Export(r.Context(), wsID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "export failed", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="workspace-`+wsID.String()+`-export.json"`)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"workspace_id": wsID,
		"tables":       tables,
	})
}
