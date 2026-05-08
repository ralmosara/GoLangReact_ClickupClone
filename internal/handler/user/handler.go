package user

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	usersvc "github.com/yourorg/clickup/internal/service/user"
)

type Handler struct{ svc *usersvc.Service }

func New(svc *usersvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) PublicRoutes(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
}

func (h *Handler) PrivateRoutes(r chi.Router) {
	r.Get("/me", h.me)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var in usersvc.RegisterInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Register(r.Context(), in)
	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, usersvc.ErrEmailTaken) {
			code = http.StatusConflict
		}
		httpx.Err(w, code, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in usersvc.LoginInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Err(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Login(r.Context(), in)
	if err != nil {
		httpx.Err(w, http.StatusUnauthorized, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := h.svc.Me(r.Context(), uid)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if u == nil {
		httpx.Err(w, http.StatusNotFound, "not found")
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}
