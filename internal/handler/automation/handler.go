package automation

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	asvc "github.com/yourorg/clickup/internal/service/automation"
)

type Handler struct {
	svc    *asvc.Service
	policy authz.Policy // optional; nil disables membership checks (legacy/dev)
}

// New builds a handler. policy may be nil — when supplied, every route
// enforces workspace (or list) membership before the service call so
// non-members can't list, create, or mutate automations they don't own.
func New(svc *asvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

// requireWorkspace wraps the policy check with a uniform JSON error response.
// Returns true if the request should continue. policy nil → allowed.
func (h *Handler) requireWorkspace(w http.ResponseWriter, r *http.Request, wsID, userID uuid.UUID) bool {
	if h.policy == nil {
		return true
	}
	if err := h.policy.RequireWorkspaceMember(r.Context(), wsID, userID); err != nil {
		if errors.Is(err, authz.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
		} else {
			httpx.Fail(w, r, http.StatusInternalServerError, "policy check failed", err)
		}
		return false
	}
	return true
}

// requireList enforces list membership. Used on the list-scoped create path
// and any handler that needs to verify the actor can act on a specific list.
func (h *Handler) requireList(w http.ResponseWriter, r *http.Request, listID, userID uuid.UUID) bool {
	if h.policy == nil {
		return true
	}
	if err := h.policy.RequireListAccess(r.Context(), listID, userID); err != nil {
		if errors.Is(err, authz.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
		} else {
			httpx.Fail(w, r, http.StatusInternalServerError, "policy check failed", err)
		}
		return false
	}
	return true
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/automations", h.listByWorkspace)
	r.Post("/workspaces/{workspaceID}/automations", h.createForWorkspace)
	r.Get("/lists/{listID}/automations", h.listByList)
	r.Post("/lists/{listID}/automations", h.createForList)
	r.Get("/automations/{id}", h.get)
	r.Patch("/automations/{id}", h.update)
	r.Delete("/automations/{id}", h.delete)
}

func (h *Handler) listByWorkspace(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	if !h.requireWorkspace(w, r, id, uid) {
		return
	}
	res, err := h.svc.ListByWorkspace(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.Automation{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) listByList(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "listID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad listID")
		return
	}
	if !h.requireList(w, r, id, uid) {
		return
	}
	res, err := h.svc.ListByList(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.Automation{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) createForWorkspace(w http.ResponseWriter, r *http.Request) {
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
	if !h.requireWorkspace(w, r, wsID, uid) {
		return
	}
	var in asvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	// Trust the URL over the body — clients sometimes echo workspace_id back
	// from the previous response and we don't want to let a body field
	// override the path-scoped check we just performed.
	in.WorkspaceID = wsID
	// If the caller picked a list scope, verify it belongs to this workspace.
	// Otherwise a workspace member could attach an automation to a list in
	// another workspace they can't see.
	if in.ListID != nil {
		if err := h.svc.VerifyListInWorkspace(r.Context(), *in.ListID, wsID); err != nil {
			httpx.Err(w, http.StatusBadRequest, "list does not belong to workspace")
			return
		}
	}
	res, err := h.svc.Create(r.Context(), uid, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) createForList(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	listID, err := uuid.Parse(chi.URLParam(r, "listID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad listID")
		return
	}
	if !h.requireList(w, r, listID, uid) {
		return
	}
	wsID, err := h.svc.WorkspaceForList(r.Context(), listID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "could not resolve workspace", err)
		return
	}
	var in asvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	in.ListID = &listID
	in.WorkspaceID = wsID
	res, err := h.svc.Create(r.Context(), uid, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	res, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}
	if !h.requireWorkspace(w, r, res.WorkspaceID, uid) {
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	// Load the existing automation first so the policy check uses the
	// real workspace, not whatever the body claims.
	existing, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if existing == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}
	if !h.requireWorkspace(w, r, existing.WorkspaceID, uid) {
		return
	}
	var in asvc.UpdateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	// If the patch tries to move scope to a different list, that list must
	// also live in the same workspace.
	if in.ListID != nil {
		if err := h.svc.VerifyListInWorkspace(r.Context(), *in.ListID, existing.WorkspaceID); err != nil {
			httpx.Err(w, http.StatusBadRequest, "list does not belong to workspace")
			return
		}
	}
	res, err := h.svc.Update(r.Context(), uid, id, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	existing, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if existing == nil {
		// Idempotent: 204 if it's already gone.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !h.requireWorkspace(w, r, existing.WorkspaceID, uid) {
		return
	}
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
