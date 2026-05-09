// Package role exposes the workspace-level role-management API. The
// permissions catalog is read-only via GET /permissions; everything else
// is gated by the role.manage permission on the target workspace.
package role

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	rolesvc "github.com/yourorg/clickup/internal/service/role"
)

type Handler struct {
	svc    *rolesvc.Service
	policy authz.Policy
}

func New(svc *rolesvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

func (h *Handler) Routes(r chi.Router) {
	// Catalog — accessible to anyone authenticated; the keys are not
	// secret, the UI needs them to render the permission matrix even
	// before the user has joined a workspace.
	r.Get("/permissions", h.listPermissions)

	// Workspace-scoped role management.
	r.Route("/workspaces/{workspaceID}/roles", func(r chi.Router) {
		r.Get("/", h.listRoles)
		r.Get("/effective", h.effectivePermissions)
		r.Post("/", h.createRole)
		r.Get("/{roleID}", h.getRole)
		r.Patch("/{roleID}", h.updateRole)
		r.Delete("/{roleID}", h.deleteRole)
		r.Put("/{roleID}/permissions", h.setRolePermissions)
		r.Post("/{roleID}/assign", h.assignToMember)
	})
}

func (h *Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.svc.ListPermissions(r.Context())
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, perms)
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	wsID, uid, ok := h.workspaceCtx(w, r)
	if !ok {
		return
	}
	if err := h.policy.RequireWorkspaceMember(r.Context(), wsID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}
	roles, err := h.svc.ListWorkspaceRoles(r.Context(), wsID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, roles)
}

func (h *Handler) effectivePermissions(w http.ResponseWriter, r *http.Request) {
	wsID, uid, ok := h.workspaceCtx(w, r)
	if !ok {
		return
	}
	// A user can always introspect their own grants — necessary for the UI
	// to disable buttons they don't have permission to use.
	perms, err := h.svc.EffectivePermissions(r.Context(), wsID, uid)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if perms == nil {
		perms = []string{}
	}
	// roles is the assigned role-name list (typically one entry like
	// "admin" or "owner"). Surfaced so the FE can gate on a strict role
	// match without having to interpret permission combinations.
	roles, err := h.svc.RoleNamesForUser(r.Context(), wsID, uid)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if roles == nil {
		roles = []string{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"workspace_id": wsID,
		"user_id":      uid,
		"permissions":  perms,
		"roles":        roles,
	})
}

type createRoleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Rank        int      `json:"rank"`
	Permissions []string `json:"permissions"`
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	wsID, uid, ok := h.workspaceCtx(w, r)
	if !ok {
		return
	}
	if err := h.policy.RequirePermission(r.Context(), wsID, uid, authz.PermRoleManage); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}
	var in createRoleInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	role, err := h.svc.Create(r.Context(), rolesvc.CreateInput{
		WorkspaceID: wsID,
		Name:        in.Name,
		Description: in.Description,
		Rank:        in.Rank,
		Permissions: in.Permissions,
	})
	if err != nil {
		h.translateErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, role)
}

func (h *Handler) getRole(w http.ResponseWriter, r *http.Request) {
	wsID, uid, ok := h.workspaceCtx(w, r)
	if !ok {
		return
	}
	if err := h.policy.RequireWorkspaceMember(r.Context(), wsID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad roleID")
		return
	}
	role, err := h.svc.GetWithPermissions(r.Context(), roleID)
	if err != nil {
		h.translateErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, role)
}

type updateRoleInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Rank        *int    `json:"rank"`
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	wsID, uid, ok := h.workspaceCtx(w, r)
	if !ok {
		return
	}
	if err := h.policy.RequirePermission(r.Context(), wsID, uid, authz.PermRoleManage); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad roleID")
		return
	}
	var in updateRoleInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	role, err := h.svc.Update(r.Context(), roleID, rolesvc.UpdateInput{
		Name:        in.Name,
		Description: in.Description,
		Rank:        in.Rank,
	})
	if err != nil {
		h.translateErr(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, role)
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	wsID, uid, ok := h.workspaceCtx(w, r)
	if !ok {
		return
	}
	if err := h.policy.RequirePermission(r.Context(), wsID, uid, authz.PermRoleManage); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad roleID")
		return
	}
	if err := h.svc.Delete(r.Context(), roleID); err != nil {
		h.translateErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type setPermissionsInput struct {
	Permissions []string `json:"permissions"`
}

func (h *Handler) setRolePermissions(w http.ResponseWriter, r *http.Request) {
	wsID, uid, ok := h.workspaceCtx(w, r)
	if !ok {
		return
	}
	if err := h.policy.RequirePermission(r.Context(), wsID, uid, authz.PermRoleManage); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad roleID")
		return
	}
	var in setPermissionsInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.SetPermissions(r.Context(), roleID, in.Permissions); err != nil {
		h.translateErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type assignInput struct {
	UserID uuid.UUID `json:"user_id"`
}

func (h *Handler) assignToMember(w http.ResponseWriter, r *http.Request) {
	wsID, uid, ok := h.workspaceCtx(w, r)
	if !ok {
		return
	}
	if err := h.policy.RequirePermission(r.Context(), wsID, uid, authz.PermMemberManageRoles); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad roleID")
		return
	}
	var in assignInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.AssignToMember(r.Context(), wsID, in.UserID, roleID); err != nil {
		h.translateErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ──────────────────────────────────────────────────────────────

func (h *Handler) workspaceCtx(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return uuid.Nil, uuid.Nil, false
	}
	return wsID, uid, true
}

func (h *Handler) translateErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, rolesvc.ErrNotFound):
		httpx.Err(w, http.StatusNotFound, "not found")
	case errors.Is(err, rolesvc.ErrBuiltinReadOnly):
		httpx.Err(w, http.StatusForbidden, "built-in roles are read-only")
	case errors.Is(err, rolesvc.ErrNameRequired):
		httpx.Err(w, http.StatusBadRequest, "name is required")
	case errors.Is(err, rolesvc.ErrInvalidPerm):
		httpx.Err(w, http.StatusBadRequest, "unknown permission key")
	default:
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
	}
}
