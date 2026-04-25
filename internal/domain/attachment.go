package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID         uuid.UUID  `json:"id"`
	TaskID     uuid.UUID  `json:"task_id"`
	UploaderID *uuid.UUID `json:"uploader_id,omitempty"`
	Filename   string     `json:"filename"`
	Path       string     `json:"path"`
	SizeBytes  int64      `json:"size_bytes"`
	MimeType   string     `json:"mime_type"`
	CreatedAt  time.Time  `json:"created_at"`
}

type AttachmentRepo interface {
	Create(ctx context.Context, a *Attachment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Attachment, error)
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]Attachment, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
