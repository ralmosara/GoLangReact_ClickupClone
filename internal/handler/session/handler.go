// Package session is the HTTP surface for the Active Sessions UI.
//
// Routes are mounted under /me/sessions so they're naturally scoped to
// the authenticated caller — there is no admin path that lets one user
// list another's sessions.
package session

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	ssvc "github.com/yourorg/clickup/internal/service/session"
)

type Handler struct{ svc *ssvc.Service }

func New(svc *ssvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/me/sessions", h.list)
	r.Delete("/me/sessions/{id}", h.revoke)
	r.Post("/me/sessions/revoke-others", h.revokeOthers)
}

type sessionView struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	UserAgent  string `json:"user_agent"`
	IP         string `json:"ip,omitempty"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at"`
	ExpiresAt  string `json:"expires_at"`
	Current    bool   `json:"current"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sessions, err := h.svc.List(r.Context(), uid)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	currentJTI, _ := middleware.SessionID(r.Context())
	out := make([]sessionView, 0, len(sessions))
	for _, s := range sessions {
		v := sessionView{
			ID:         s.ID.String(),
			Label:      s.Label,
			UserAgent:  s.UserAgent,
			CreatedAt:  s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			LastSeenAt: s.LastSeenAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			ExpiresAt:  s.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			Current:    s.ID == currentJTI,
		}
		if s.IP != nil {
			v.IP = s.IP.String()
		}
		out = append(out, v)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	jti, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := h.svc.Revoke(r.Context(), uid, jti); err != nil {
		if errors.Is(err, ssvc.ErrNotOwner) {
			httpx.Err(w, http.StatusForbidden, "not your session")
			return
		}
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) revokeOthers(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	keep, ok := middleware.SessionID(r.Context())
	if !ok {
		// Pre-sessions tokens have no jti to keep — refuse rather than
		// nuke every session for the user (which would include theirs).
		httpx.Err(w, http.StatusConflict, "current session is not tracked; sign in again to enable per-device sign-out")
		return
	}
	n, err := h.svc.RevokeOthers(r.Context(), uid, keep)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]int{"revoked": n})
}
