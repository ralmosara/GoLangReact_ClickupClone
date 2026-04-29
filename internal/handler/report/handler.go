package report

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	repsvc "github.com/yourorg/clickup/internal/service/report"
)

type Handler struct{ svc *repsvc.Service }

func New(svc *repsvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/workspaces/{workspaceID}/reports/accomplishments", h.accomplishments)
	r.Get("/workspaces/{workspaceID}/reports/accomplishments.pdf", h.accomplishmentsPDF)
	r.Get("/workspaces/{workspaceID}/reports/accomplishments.xlsx", h.accomplishmentsXLSX)
}

// parseInput pulls the (workspaceID, userID, from, to, group_by) tuple shared
// by the JSON list and the export endpoints.
func (h *Handler) parseInput(r *http.Request) (repsvc.AccomplishmentsInput, *httpErr) {
	uid, ok := middleware.UserID(r.Context())
	if !ok {
		return repsvc.AccomplishmentsInput{}, &httpErr{http.StatusUnauthorized, "unauthorized"}
	}
	wid, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
	if err != nil {
		return repsvc.AccomplishmentsInput{}, &httpErr{http.StatusBadRequest, "bad workspace id"}
	}

	q := r.URL.Query()
	now := time.Now().UTC()
	to := now
	from := now.AddDate(0, 0, -30)

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return repsvc.AccomplishmentsInput{}, &httpErr{http.StatusBadRequest, "bad from"}
		}
		from = t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return repsvc.AccomplishmentsInput{}, &httpErr{http.StatusBadRequest, "bad to"}
		}
		to = t
	}

	groupBy := repsvc.GroupByDay
	switch q.Get("group_by") {
	case "week":
		groupBy = repsvc.GroupByWeek
	case "", "day":
		groupBy = repsvc.GroupByDay
	default:
		return repsvc.AccomplishmentsInput{}, &httpErr{http.StatusBadRequest, "group_by must be 'day' or 'week'"}
	}

	return repsvc.AccomplishmentsInput{
		WorkspaceID: wid,
		UserID:      uid,
		From:        from,
		To:          to,
		GroupBy:     groupBy,
	}, nil
}

type httpErr struct {
	code int
	msg  string
}

func (h *Handler) accomplishments(w http.ResponseWriter, r *http.Request) {
	in, herr := h.parseInput(r)
	if herr != nil {
		httpx.Err(w, herr.code, herr.msg)
		return
	}
	res, err := h.svc.Accomplishments(r.Context(), in)
	if err != nil {
		if errors.Is(err, repsvc.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res == nil {
		res = []domain.AccomplishmentBucket{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) accomplishmentsPDF(w http.ResponseWriter, r *http.Request) {
	in, herr := h.parseInput(r)
	if herr != nil {
		httpx.Err(w, herr.code, herr.msg)
		return
	}
	body, err := h.svc.ExportPDF(r.Context(), in)
	if err != nil {
		if errors.Is(err, repsvc.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, exportFilename(in, "pdf")))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	_, _ = w.Write(body)
}

func (h *Handler) accomplishmentsXLSX(w http.ResponseWriter, r *http.Request) {
	in, herr := h.parseInput(r)
	if herr != nil {
		httpx.Err(w, herr.code, herr.msg)
		return
	}
	body, err := h.svc.ExportXLSX(r.Context(), in)
	if err != nil {
		if errors.Is(err, repsvc.ErrForbidden) {
			httpx.Err(w, http.StatusForbidden, "forbidden")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, exportFilename(in, "xlsx")))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	_, _ = w.Write(body)
}

func exportFilename(in repsvc.AccomplishmentsInput, ext string) string {
	return fmt.Sprintf("accomplishments_%s_%s_to_%s.%s",
		in.GroupBy,
		in.From.Format("2006-01-02"),
		in.To.Format("2006-01-02"),
		ext,
	)
}
