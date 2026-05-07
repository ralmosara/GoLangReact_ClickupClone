package folder

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	fsvc "github.com/yourorg/clickup/internal/service/folder"
)

type Handler struct {
	svc    *fsvc.Service
	policy authz.Policy
}

func New(svc *fsvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/folders", h.create)
	r.Get("/spaces/{spaceID}/folders", h.listBySpace)
	r.Delete("/folders/{id}", h.delete)
}

func actor(r *http.Request) (uuid.UUID, bool) { return middleware.UserID(r.Context()) }

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in fsvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.policy.RequireSpaceMember(r.Context(), in.SpaceID, uid); err != nil {
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

func (h *Handler) listBySpace(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "spaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad space id")
		return
	}

	if err := h.policy.RequireSpaceMember(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.List(r.Context(), id)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
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

	// Resolve space from folder to check permissions
	f, err := h.svc.Get(r.Context(), id)
	if err != nil || f == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}

	if err := h.policy.RequireSpaceRole(r.Context(), f.SpaceID, uid, authz.RoleAdmin); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
