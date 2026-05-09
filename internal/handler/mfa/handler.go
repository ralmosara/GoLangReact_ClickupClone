// Package mfa exposes the MFA enrolment, verification, and disabling
// endpoints. The actual login challenge handshake lives in the user
// handler — this package only manages enrolment from an authenticated
// session.
package mfa

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	mfasvc "github.com/yourorg/clickup/internal/service/mfa"
)

type Handler struct{ svc *mfasvc.Service }

func New(svc *mfasvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Route("/me/mfa", func(r chi.Router) {
		r.Get("/status", h.status)
		r.Post("/enroll", h.enroll)
		r.Post("/verify-enroll", h.verifyEnroll)
		r.Post("/recovery-codes", h.regenerateRecovery)
		r.Delete("/", h.disable)
	})
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	enrolled, err := h.svc.IsEnrolled(r.Context(), uid)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	remaining := 0
	if enrolled {
		remaining, _ = h.svc.RemainingRecoveryCodes(r.Context(), uid)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enrolled":                  enrolled,
		"remaining_recovery_codes":  remaining,
	})
}

func (h *Handler) enroll(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	res, err := h.svc.Enroll(r.Context(), uid)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) verifyEnroll(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.VerifyEnrollment(r.Context(), uid, in.Code)
	if err != nil {
		switch {
		case errors.Is(err, mfasvc.ErrInvalidCode):
			httpx.Err(w, http.StatusBadRequest, "invalid code")
		case errors.Is(err, mfasvc.ErrAlreadyEnabled):
			httpx.Err(w, http.StatusConflict, "already enrolled")
		case errors.Is(err, mfasvc.ErrNotEnrolled):
			httpx.Err(w, http.StatusBadRequest, "no enrolment in progress; call POST /me/mfa/enroll first")
		default:
			httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		}
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) regenerateRecovery(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	codes, err := h.svc.RegenerateRecoveryCodes(r.Context(), uid)
	if err != nil {
		if errors.Is(err, mfasvc.ErrNotEnrolled) {
			httpx.Err(w, http.StatusBadRequest, "mfa not enabled")
			return
		}
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

func (h *Handler) disable(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.svc.Disable(r.Context(), uid); err != nil {
		if errors.Is(err, mfasvc.ErrNotEnrolled) {
			httpx.Err(w, http.StatusBadRequest, "mfa not enabled")
			return
		}
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
