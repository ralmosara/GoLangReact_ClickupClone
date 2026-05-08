package dependency

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	dsvc "github.com/yourorg/clickup/internal/service/dependency"
)

type Handler struct{ svc *dsvc.Service }

func New(svc *dsvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/tasks/{taskID}/dependencies", h.graph)
	r.Post("/tasks/{taskID}/dependencies", h.create)
	r.Delete("/tasks/{taskID}/dependencies/{dependsOnID}", h.delete)
}

func (h *Handler) graph(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad taskID")
		return
	}
	res, err := h.svc.Graph(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad taskID")
		return
	}
	var in dsvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Create(r.Context(), uid, taskID, in)
	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, dsvc.ErrCycleDetected) {
			code = http.StatusConflict
		}
		httpx.Err(w, code, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad taskID")
		return
	}
	depID, err := uuid.Parse(chi.URLParam(r, "dependsOnID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad dependsOnID")
		return
	}
	if err := h.svc.Delete(r.Context(), uid, taskID, depID); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
