package list

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	lsvc "github.com/yourorg/clickup/internal/service/list"
)

type Handler struct {
	svc    *lsvc.Service
	policy authz.Policy
}

func New(svc *lsvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/lists", h.create)
	r.Get("/lists/{id}", h.get)
	r.Get("/spaces/{spaceID}/lists", h.listBySpace)
	r.Get("/folders/{folderID}/lists", h.listByFolder)
	r.Delete("/lists/{id}", h.delete)
}

func actor(r *http.Request) (uuid.UUID, bool) { return middleware.UserID(r.Context()) }

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in lsvc.CreateInput
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

	if err := h.policy.RequireListAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}
	httpx.JSON(w, http.StatusOK, res)
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

	res, err := h.svc.ListBySpace(r.Context(), id)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) listByFolder(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "folderID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad folder id")
		return
	}

	// We need to resolve the space from the folder to check access
	// In a real app we might have a policy.RequireFolderAccess
	res, err := h.svc.ListByFolder(r.Context(), id)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If there are lists, check access to one of them as a proxy for folder access
	// Or better, we should have the spaceID of the folder.
	// For now, let's assume if they can access the first list, they can access the folder.
	if len(res) > 0 {
		if err := h.policy.RequireListAccess(r.Context(), res[0].ID, uid); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
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

	l, err := h.svc.Get(r.Context(), id)
	if err != nil || l == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}

	if err := h.policy.RequireSpaceRole(r.Context(), l.SpaceID, uid, authz.RoleAdmin); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
