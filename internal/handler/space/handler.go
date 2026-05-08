package space

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	spsvc "github.com/yourorg/clickup/internal/service/space"
)

type Handler struct {
	svc    *spsvc.Service
	policy authz.Policy
}

func New(svc *spsvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/spaces", h.create)
	r.Get("/workspaces/{workspaceID}/spaces", h.listByWorkspace)
	r.Get("/spaces/{id}", h.get)
	r.Patch("/spaces/{id}", h.update)
	r.Delete("/spaces/{id}", h.delete)
}

func actor(r *http.Request) (uuid.UUID, bool) { return middleware.UserID(r.Context()) }

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in spsvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.policy.RequireWorkspaceMember(r.Context(), in.WorkspaceID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) listByWorkspace(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wid, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspace id")
		return
	}

	if err := h.policy.RequireWorkspaceMember(r.Context(), wid, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.List(r.Context(), wid)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}

	if err := h.policy.RequireSpaceMember(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
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
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}

	if err := h.policy.RequireSpaceMember(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	sp, err := h.svc.Get(r.Context(), id)
	if err != nil || sp == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}
	if err := httpx.Decode(r, sp); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	sp.ID = id
	if err := h.svc.Update(r.Context(), sp); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, sp)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}

	if err := h.policy.RequireSpaceRole(r.Context(), id, uid, authz.RoleAdmin); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
