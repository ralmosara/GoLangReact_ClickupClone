package milestone

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	msvc "github.com/yourorg/clickup/internal/service/milestone"
)

type Handler struct{ svc *msvc.Service }

func New(svc *msvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/milestones", h.list)
	r.Post("/workspaces/{workspaceID}/milestones", h.create)
	r.Get("/milestones/{id}", h.get)
	r.Patch("/milestones/{id}", h.update)
	r.Delete("/milestones/{id}", h.delete)
	r.Get("/milestones/{id}/tasks", h.listTasks)
	r.Post("/milestones/{id}/tasks/{taskID}", h.attach)
	r.Delete("/milestones/{id}/tasks/{taskID}", h.detach)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	res, err := h.svc.List(r.Context(), wsID)
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
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	var in msvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	in.WorkspaceID = wsID
	res, err := h.svc.Create(r.Context(), uid, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	res, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, msvc.ErrNotFound) {
			httpx.Err(w, http.StatusNotFound, "not found")
			return
		}
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
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
	var in msvc.UpdateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Update(r.Context(), uid, id, in)
	if err != nil {
		if errors.Is(err, msvc.ErrNotFound) {
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	res, err := h.svc.ListTasks(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []uuid.UUID{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"task_ids": res})
}

func (h *Handler) attach(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	mID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad milestone id")
		return
	}
	tID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad task id")
		return
	}
	if err := h.svc.AttachTask(r.Context(), uid, mID, tID); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) detach(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	mID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad milestone id")
		return
	}
	tID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad task id")
		return
	}
	if err := h.svc.DetachTask(r.Context(), uid, mID, tID); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
