package member

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	msvc "github.com/yourorg/clickup/internal/service/member"
)

type Handler struct{ svc *msvc.Service }

func New(svc *msvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/members", h.listWS)
	r.Post("/workspaces/{workspaceID}/members", h.addWS)
	r.Delete("/workspaces/{workspaceID}/members/{userID}", h.removeWS)

	r.Get("/spaces/{spaceID}/members", h.listSpace)
	r.Post("/spaces/{spaceID}/members", h.addSpace)
	r.Delete("/spaces/{spaceID}/members/{userID}", h.removeSpace)
}

// --- workspace --------------------------------------------------------------

func (h *Handler) listWS(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	res, err := h.svc.ListWorkspaceMembers(r.Context(), wsID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.Member{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) addWS(w http.ResponseWriter, r *http.Request) {
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
	var in msvc.AddInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.AddWorkspaceMember(r.Context(), uid, wsID, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) removeWS(w http.ResponseWriter, r *http.Request) {
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
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad userID")
		return
	}
	if err := h.svc.RemoveWorkspaceMember(r.Context(), uid, wsID, userID); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- space -------------------------------------------------------------------

func (h *Handler) listSpace(w http.ResponseWriter, r *http.Request) {
	spaceID, err := uuid.Parse(chi.URLParam(r, "spaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad spaceID")
		return
	}
	res, err := h.svc.ListSpaceMembers(r.Context(), spaceID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.Member{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) addSpace(w http.ResponseWriter, r *http.Request) {
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
	var in msvc.AddInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.AddSpaceMember(r.Context(), uid, spaceID, in)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) removeSpace(w http.ResponseWriter, r *http.Request) {
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
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad userID")
		return
	}
	if err := h.svc.RemoveSpaceMember(r.Context(), uid, spaceID, userID); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
