package comment

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	csvc "github.com/yourorg/clickup/internal/service/comment"
)

type Handler struct {
	svc    *csvc.Service
	policy authz.Policy
}

func New(svc *csvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/comments", h.create)
	r.Get("/tasks/{taskID}/comments", h.listByTask)
	r.Delete("/comments/{id}", h.delete)
}

func actor(r *http.Request) (uuid.UUID, bool) { return middleware.UserID(r.Context()) }

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in csvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.policy.RequireTaskAccess(r.Context(), in.TaskID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.Create(r.Context(), uid, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) listByTask(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad task id")
		return
	}

	if err := h.policy.RequireTaskAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.ListByTask(r.Context(), id)
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

	// Resolve task from comment to check permissions
	c, err := h.svc.Get(r.Context(), id)
	if err != nil || c == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}

	if err := h.policy.RequireTaskAccess(r.Context(), c.TaskID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
