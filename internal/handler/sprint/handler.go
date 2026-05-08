package sprint

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	ssvc "github.com/yourorg/clickup/internal/service/sprint"
)

type Handler struct{ svc *ssvc.Service }

func New(svc *ssvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/lists/{listID}/sprints", h.listByList)
	r.Post("/lists/{listID}/sprints", h.createForList)
	r.Get("/sprints/{id}", h.get)
	r.Patch("/sprints/{id}", h.update)
	r.Delete("/sprints/{id}", h.delete)
	r.Post("/sprints/{id}/close", h.close)
	r.Get("/sprints/{id}/burndown", h.burndown)
}

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
		res = []domain.Sprint{}
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
	var in ssvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	in.ListID = listID
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
	var in ssvc.UpdateInput
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

func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
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
	res, err := h.svc.Close(r.Context(), uid, id)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) burndown(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	res, err := h.svc.Burndown(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.BurndownPoint{}
	}
	httpx.JSON(w, http.StatusOK, res)
}
