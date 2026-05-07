package view

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	vsvc "github.com/yourorg/clickup/internal/service/view"
)

type Handler struct {
	svc    *vsvc.Service
	policy authz.Policy
}

func New(svc *vsvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/lists/{listID}/views", h.listByList)
	r.Post("/lists/{listID}/views", h.createForList)

	r.Get("/spaces/{spaceID}/views", h.listBySpace)
	r.Post("/spaces/{spaceID}/views", h.createForSpace)

	r.Get("/views/{id}", h.get)
	r.Patch("/views/{id}", h.update)
	r.Delete("/views/{id}", h.delete)
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

	if err := h.policy.RequireListAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.ListByList(r.Context(), id)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res == nil {
		res = []domain.View{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) listBySpace(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "spaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad spaceID")
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
	if res == nil {
		res = []domain.View{}
	}
	httpx.JSON(w, http.StatusOK, res)
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

	if err := h.policy.RequireListAccess(r.Context(), listID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	var in vsvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	in.ListID = &listID
	res, err := h.svc.Create(r.Context(), uid, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) createForSpace(w http.ResponseWriter, r *http.Request) {
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

	if err := h.policy.RequireSpaceMember(r.Context(), spaceID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	var in vsvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	in.SpaceID = &spaceID
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
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}

	if res.ListID != nil {
		if err := h.policy.RequireListAccess(r.Context(), *res.ListID, uid); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
	} else if res.SpaceID != nil {
		if err := h.policy.RequireSpaceMember(r.Context(), *res.SpaceID, uid); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
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

	view, err := h.svc.Get(r.Context(), id)
	if err != nil || view == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}

	if view.ListID != nil {
		if err := h.policy.RequireListAccess(r.Context(), *view.ListID, uid); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
	} else if view.SpaceID != nil {
		if err := h.policy.RequireSpaceMember(r.Context(), *view.SpaceID, uid); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	var in vsvc.UpdateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
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

	view, err := h.svc.Get(r.Context(), id)
	if err != nil || view == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}

	if view.ListID != nil {
		if err := h.policy.RequireListAccess(r.Context(), *view.ListID, uid); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
	} else if view.SpaceID != nil {
		if err := h.policy.RequireSpaceMember(r.Context(), *view.SpaceID, uid); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
