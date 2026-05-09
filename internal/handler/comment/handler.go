package comment

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	csvc "github.com/yourorg/clickup/internal/service/comment"
)

type Handler struct{ svc *csvc.Service }

func New(svc *csvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Post("/comments", h.create)
	r.Get("/tasks/{taskID}/comments", h.listByTask)
	r.Get("/comments/{id}/replies", h.listReplies)
	r.Patch("/comments/{id}", h.update)
	r.Delete("/comments/{id}", h.delete)
	// Reactions — POST adds, DELETE removes. Emoji is in the body so
	// it stays UTF-safe (URL-encoding pile-of-poo is doable but ugly).
	r.Post("/comments/{id}/reactions", h.react)
	r.Delete("/comments/{id}/reactions", h.unreact)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in csvc.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
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
	uid, _ := middleware.UserID(r.Context()) // viewer for "did I react"
	id, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad task id")
		return
	}
	res, err := h.svc.ListByTask(r.Context(), uid, id)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) listReplies(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	res, err := h.svc.ListReplies(r.Context(), uid, id)
	if err != nil {
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
	var in struct{ Body string `json:"body"` }
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Update(r.Context(), uid, id, in.Body)
	if err != nil {
		switch {
		case errors.Is(err, csvc.ErrNotAuthor):
			httpx.Err(w, http.StatusForbidden, "only the author can edit")
		case errors.Is(err, csvc.ErrCommentDeleted):
			httpx.Err(w, http.StatusConflict, "comment is deleted")
		default:
			httpx.Err(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// delete now soft-deletes by default — preserves thread structure. The
// hard-delete path is reserved for admin sweeps and isn't surfaced over
// HTTP.
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
	if err := h.svc.SoftDelete(r.Context(), uid, id); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) react(w http.ResponseWriter, r *http.Request) {
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
	var in struct{ Emoji string `json:"emoji"` }
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.React(r.Context(), uid, id, in.Emoji); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) unreact(w http.ResponseWriter, r *http.Request) {
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
	var in struct{ Emoji string `json:"emoji"` }
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.Unreact(r.Context(), uid, id, in.Emoji); err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
