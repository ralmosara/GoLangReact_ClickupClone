package member

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	msvc "github.com/yourorg/clickup/internal/service/member"
)

type Handler struct {
	svc    *msvc.Service
	policy authz.Policy // optional; if nil, the rank-based fallback is used (legacy behaviour)
}

// New constructs the handler without policy. Equivalent to the legacy
// behaviour where the route was protected only by workspace membership.
// Prefer NewWithPolicy in main.go so admin-driven invites are gated by
// the new permission catalog.
func New(svc *msvc.Service) *Handler { return &Handler{svc: svc} }

// NewWithPolicy is the post-RBAC constructor. addWS/removeWS now require
// the member.invite / member.remove permissions instead of just workspace
// membership.
func NewWithPolicy(svc *msvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/members", h.listWS)
	r.Get("/workspaces/{workspaceID}/members/lookup", h.lookupWS)
	r.Post("/workspaces/{workspaceID}/members", h.addWS)
	r.Delete("/workspaces/{workspaceID}/members/{userID}", h.removeWS)

	r.Get("/spaces/{spaceID}/members", h.listSpace)
	r.Post("/spaces/{spaceID}/members", h.addSpace)
	r.Delete("/spaces/{spaceID}/members/{userID}", h.removeSpace)
}

// --- workspace --------------------------------------------------------------

func (h *Handler) listWS(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	res, err := h.svc.ListWorkspaceMembers(r.Context(), wsID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.Member{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

// lookupWS resolves an email to {exists, already_member, name} so the
// frontend invite form can adapt before submission. Same authorization
// as the invite itself (member.invite on the workspace).
func (h *Handler) lookupWS(w http.ResponseWriter, r *http.Request) {
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
	if h.policy != nil {
		if err := h.policy.RequirePermission(r.Context(), wsID, uid, authz.PermMemberInvite); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
	}
	email := r.URL.Query().Get("email")
	if email == "" {
		httpx.Err(w, http.StatusBadRequest, "email query param required")
		return
	}
	res, err := h.svc.LookupForInvite(r.Context(), wsID, email)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) addWS(w http.ResponseWriter, r *http.Request) {
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
	if h.policy != nil {
		if err := h.policy.RequirePermission(r.Context(), wsID, uid, authz.PermMemberInvite); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
	}
	var in msvc.AddInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.AddWorkspaceMember(r.Context(), uid, wsID, in)
	if err != nil {
		switch {
		case errors.Is(err, msvc.ErrUserNotFound):
			// Surface the actionable hint so the SPA can render a
			// "Create them in User Management" deep link without having
			// to parse a generic error string.
			httpx.JSON(w, http.StatusNotFound, map[string]string{
				"error":  "no user with that email — create them in User Management first",
				"reason": "user_not_found",
			})
		default:
			httpx.Err(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) removeWS(w http.ResponseWriter, r *http.Request) {
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
	if h.policy != nil {
		if err := h.policy.RequirePermission(r.Context(), wsID, uid, authz.PermMemberRemove); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad userID")
		return
	}
	if err := h.svc.RemoveWorkspaceMember(r.Context(), uid, wsID, userID); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- space -------------------------------------------------------------------

func (h *Handler) listSpace(w http.ResponseWriter, r *http.Request) {
	spaceID, err := uuid.Parse(chi.URLParam(r, "spaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad spaceID")
		return
	}
	res, err := h.svc.ListSpaceMembers(r.Context(), spaceID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.Member{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) addSpace(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	spaceID, err := uuid.Parse(chi.URLParam(r, "spaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad spaceID")
		return
	}
	var in msvc.AddInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.AddSpaceMember(r.Context(), uid, spaceID, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) removeSpace(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	spaceID, err := uuid.Parse(chi.URLParam(r, "spaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad spaceID")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad userID")
		return
	}
	if err := h.svc.RemoveSpaceMember(r.Context(), uid, spaceID, userID); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
