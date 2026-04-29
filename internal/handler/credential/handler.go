package credential

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	credsvc "github.com/yourorg/clickup/internal/service/credential"
)

type Handler struct{ svc *credsvc.Service }

func New(svc *credsvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Post("/workspaces/{workspaceID}/credentials", h.create)
	r.Get("/workspaces/{workspaceID}/credentials", h.list)
	r.Get("/workspaces/{workspaceID}/credentials/{id}", h.get)
	r.Patch("/workspaces/{workspaceID}/credentials/{id}", h.update)
	r.Delete("/workspaces/{workspaceID}/credentials/{id}", h.delete)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wid, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspace id")
		return
	}
	var in credsvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Create(r.Context(), wid, uid, in)
	if err != nil {
		if errors.Is(err, credsvc.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wid, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspace id")
		return
	}
	res, err := h.svc.List(r.Context(), wid, uid)
	if err != nil {
		if errors.Is(err, credsvc.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wid, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspace id")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	res, err := h.svc.Get(r.Context(), wid, uid, id)
	if err != nil {
		if errors.Is(err, credsvc.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, credsvc.ErrNotFound) {
			httpx.Err(w, http.StatusNotFound, "not found")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, err.Error())
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
	wid, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspace id")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	var in credsvc.UpdateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Update(r.Context(), wid, uid, id, in)
	if err != nil {
		if errors.Is(err, credsvc.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, credsvc.ErrNotFound) {
			httpx.Err(w, http.StatusNotFound, "not found")
			return
		}
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
	wid, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspace id")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := h.svc.Delete(r.Context(), wid, uid, id); err != nil {
		if errors.Is(err, credsvc.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
