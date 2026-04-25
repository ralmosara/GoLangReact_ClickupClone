package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Form field kinds the builder supports. Keep in sync with the frontend enum.
const (
	FormFieldText     = "text"
	FormFieldTextarea = "textarea"
	FormFieldNumber   = "number"
	FormFieldEmail    = "email"
	FormFieldSelect   = "select"
	FormFieldCheckbox = "checkbox"
)

type Form struct {
	ID          uuid.UUID       `json:"id"`
	ListID      uuid.UUID       `json:"list_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Fields      json.RawMessage `json:"fields"`
	IsPublic    bool            `json:"is_public"`
	SubmitCount int             `json:"submit_count"`
	CreatorID   *uuid.UUID      `json:"creator_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type FormSubmission struct {
	ID          uuid.UUID       `json:"id"`
	FormID      uuid.UUID       `json:"form_id"`
	TaskID      *uuid.UUID      `json:"task_id,omitempty"`
	Payload     json.RawMessage `json:"payload"`
	SubmittedAt time.Time       `json:"submitted_at"`
	SubmitterIP string          `json:"submitter_ip,omitempty"`
}

type FormRepo interface {
	Create(ctx context.Context, f *Form) error
	GetByID(ctx context.Context, id uuid.UUID) (*Form, error)
	ListByList(ctx context.Context, listID uuid.UUID) ([]Form, error)
	Update(ctx context.Context, f *Form) error
	Delete(ctx context.Context, id uuid.UUID) error

	RecordSubmission(ctx context.Context, sub *FormSubmission) error
	ListSubmissions(ctx context.Context, formID uuid.UUID, limit int) ([]FormSubmission, error)
}
