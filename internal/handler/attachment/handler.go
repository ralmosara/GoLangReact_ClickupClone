package attachment

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/httpx"
	"github.com/yourorg/clickup/internal/middleware"
	asvc "github.com/yourorg/clickup/internal/service/attachment"
)

const defaultMaxUpload = 25 << 20 // 25 MiB per file when not configured

type Handler struct {
	svc      *asvc.Service
	maxBytes int64
}

func New(svc *asvc.Service, maxBytes int64) *Handler {
	if maxBytes <= 0 {
		maxBytes = defaultMaxUpload
	}
	return &Handler{svc: svc, maxBytes: maxBytes}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/tasks/{taskID}/attachments", h.upload)
	r.Get("/tasks/{taskID}/attachments", h.list)
	r.Get("/attachments/{id}", h.download)
	r.Delete("/attachments/{id}", h.delete)
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
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

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes)
	if err := r.ParseMultipartForm(h.maxBytes); err != nil {
		httpx.Err(w, http.StatusRequestEntityTooLarge, "file too large")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "missing 'file'")
		return
	}
	defer file.Close()

	mime := header.Header.Get("Content-Type")
	att, err := h.svc.Upload(r.Context(), asvc.UploadInput{
		TaskID:   taskID,
		Uploader: uid,
		Filename: header.Filename,
		MimeType: mime,
		Body:     file,
	})
	if err != nil {
		if errors.Is(err, asvc.ErrUnsupportedFileType) {
			httpx.Err(w, http.StatusBadRequest, "unsupported file type")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, "upload failed")
		return
	}
	httpx.JSON(w, http.StatusCreated, att)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad taskID")
		return
	}
	res, err := h.svc.ListByTask(r.Context(), taskID)
	if err != nil {
		httpx.Fail(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}
	if res == nil {
		res = []domain.Attachment{}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "bad id")
		return
	}
	att, rc, err := h.svc.Open(r.Context(), id)
	if err != nil {
		httpx.Err(w, http.StatusNotFound, err.Error())
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", att.MimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if att.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(att.SizeBytes, 10))
	}
	// Force a download disposition. Inline rendering of user-uploaded content
	// is the stored-XSS surface we just closed via sniffing — keep it closed.
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(att.Filename))
	_, _ = io.Copy(w, rc)
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
