package customfield

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	cfsvc "github.com/yourorg/clickup/internal/service/customfield"
)

type Handler struct{ svc *cfsvc.Service }

func New(svc *cfsvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/lists/{listID}/custom-fields", h.listByList)
	r.Post("/lists/{listID}/custom-fields", h.createForList)

	r.Get("/workspaces/{workspaceID}/custom-fields", h.listByWorkspace)
	r.Post("/workspaces/{workspaceID}/custom-fields", h.createForWorkspace)

	r.Patch("/custom-fields/{id}", h.update)
	r.Delete("/custom-fields/{id}", h.delete)

	r.Get("/tasks/{taskID}/custom-values", h.listValues)
	r.Put("/tasks/{taskID}/custom-values", h.upsertValue)
	r.Delete("/tasks/{taskID}/custom-values/{fieldID}", h.deleteValue)
}

// --- fields ------------------------------------------------------------------

func (h *Handler) listByList(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "listID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad listID")
		return
	}
	res, err := h.svc.ListByList(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.CustomField{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) listByWorkspace(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	res, err := h.svc.ListByWorkspace(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.CustomField{}
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
	var in cfsvc.FieldCreateInput
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

func (h *Handler) createForWorkspace(w http.ResponseWriter, r *http.Request) {
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
	var in cfsvc.FieldCreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	in.WorkspaceID = wsID
	in.ListID = nil
	res, err := h.svc.Create(r.Context(), uid, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
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
	var in cfsvc.FieldUpdateInput
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
	if err := h.svc.Delete(r.Context(), uid, id); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- values ------------------------------------------------------------------

func (h *Handler) listValues(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad taskID")
		return
	}
	res, err := h.svc.ListValues(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.CustomValue{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) upsertValue(w http.ResponseWriter, r *http.Request) {
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
	var in cfsvc.ValueInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.UpsertValue(r.Context(), uid, taskID, in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteValue(w http.ResponseWriter, r *http.Request) {
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
	fieldID, err := uuid.Parse(chi.URLParam(r, "fieldID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad fieldID")
		return
	}
	if err := h.svc.DeleteValue(r.Context(), uid, taskID, fieldID); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
