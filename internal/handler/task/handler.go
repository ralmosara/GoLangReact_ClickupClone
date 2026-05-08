package task

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/authz"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	tsvc "github.com/yourorg/clickup/internal/service/task"
)

type Handler struct {
	svc    *tsvc.Service
	policy authz.Policy
}

func New(svc *tsvc.Service, policy authz.Policy) *Handler {
	return &Handler{svc: svc, policy: policy}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/tasks", h.create)
	r.Get("/tasks", h.list)
	r.Get("/tasks/{id}", h.get)
	r.Patch("/tasks/{id}", h.update)
	r.Delete("/tasks/{id}", h.delete)

	r.Patch("/tasks/{id}/position", h.reorder)
	r.Post("/tasks/{id}/subtasks", h.createSubtask)
	r.Get("/tasks/{id}/subtasks", h.listSubtasks)

	r.Post("/tasks/{id}/archive", h.archive)
	r.Post("/tasks/{id}/unarchive", h.unarchive)

	r.Get("/tasks/{id}/assignees", h.listAssignees)
	r.Post("/tasks/{id}/assignees", h.addAssignee)
	r.Delete("/tasks/{id}/assignees/{userID}", h.removeAssignee)
}

func actor(r *http.Request) (uuid.UUID, bool) { return middleware.UserID(r.Context()) }

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in tsvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.policy.RequireListAccess(r.Context(), in.ListID, uid); err != nil {
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

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	f := domain.TaskFilter{}
	q := r.URL.Query()
	if v := q.Get("list_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.Err(w, http.StatusBadRequest, "bad list_id")
			return
		}
		if err := h.policy.RequireListAccess(r.Context(), id, uid); err != nil {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
		f.ListID = &id
	}
	if v := q.Get("assignee_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.Err(w, http.StatusBadRequest, "bad assignee_id")
			return
		}
		f.AssigneeID = &id
	}
	if v := q.Get("status"); v != "" {
		f.Status = &v
	}
	if v := q.Get("status_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.Err(w, http.StatusBadRequest, "bad status_id")
			return
		}
		f.StatusID = &id
	}
	if q.Get("include_subs") == "true" {
		f.IncludeSubs = true
	}
	if v := q.Get("archived"); v != "" {
		b := v == "true" || v == "1"
		f.Archived = &b
	}
	res, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.Task{}
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

	if err := h.policy.RequireTaskAccess(r.Context(), id, uid); err != nil {
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

	if err := h.policy.RequireTaskAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	var in tsvc.UpdateInput
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

func (h *Handler) reorder(w http.ResponseWriter, r *http.Request) {
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

	if err := h.policy.RequireTaskAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	var in tsvc.ReorderInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Reorder(r.Context(), uid, id, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
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

	if err := h.policy.RequireTaskAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createSubtask(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	parentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}

	if err := h.policy.RequireTaskAccess(r.Context(), parentID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	var in tsvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	in.ParentTaskID = &parentID
	if in.ListID == uuid.Nil {
		parent, err := h.svc.Get(r.Context(), parentID)
		if err != nil || parent == nil {
			httpx.Err(w, http.StatusBadRequest, "parent not found")
			return
		}
		in.ListID = parent.ListID
	}
	res, err := h.svc.Create(r.Context(), uid, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
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

	if err := h.policy.RequireTaskAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.Archive(r.Context(), uid, id)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) unarchive(w http.ResponseWriter, r *http.Request) {
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

	if err := h.policy.RequireTaskAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.Unarchive(r.Context(), uid, id)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) listSubtasks(w http.ResponseWriter, r *http.Request) {
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

	if err := h.policy.RequireTaskAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.ListSubtasks(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.Task{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) listAssignees(w http.ResponseWriter, r *http.Request) {
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

	if err := h.policy.RequireTaskAccess(r.Context(), id, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	res, err := h.svc.ListAssignees(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []uuid.UUID{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

type assigneeInput struct {
	UserID uuid.UUID `json:"user_id"`
}

func (h *Handler) addAssignee(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}

	if err := h.policy.RequireTaskAccess(r.Context(), taskID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	var in assigneeInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.AddAssignee(r.Context(), uid, taskID, in.UserID); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeAssignee(w http.ResponseWriter, r *http.Request) {
	uid, ok := actor(r)
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}

	if err := h.policy.RequireTaskAccess(r.Context(), taskID, uid); err != nil {
		httpx.Err(w, http.StatusForbidden, "forbidden")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad userID")
		return
	}
	if err := h.svc.RemoveAssignee(r.Context(), uid, taskID, userID); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
