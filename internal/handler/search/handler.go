package search

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	ssvc "github.com/yourorg/clickup/internal/service/search"
)

type Handler struct{ svc *ssvc.Service }

func New(svc *ssvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/search", h.search)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad workspaceID")
		return
	}
	q := r.URL.Query()
	query := q.Get("q")
	limit, _ := strconv.Atoi(q.Get("limit"))
	res, err := h.svc.Search(r.Context(), wsID, query, limit)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.SearchHit{}
	}
	httpx.JSON(w, http.StatusOK, res)
}
