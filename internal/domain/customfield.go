package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Custom field kinds. Frontend renders the matching input; backend only
// validates that the kind is known and the config/value shapes round-trip JSON.
const (
	FieldKindText     = "text"
	FieldKindNumber   = "number"
	FieldKindDropdown = "dropdown"
	FieldKindLabels   = "labels"
	FieldKindDate     = "date"
	FieldKindCheckbox = "checkbox"
	FieldKindURL      = "url"
	FieldKindEmail    = "email"
	FieldKindPhone    = "phone"
	FieldKindMoney    = "money"
	FieldKindProgress = "progress"
	FieldKindPeople   = "people"
)

var validFieldKinds = map[string]bool{
	FieldKindText: true, FieldKindNumber: true, FieldKindDropdown: true, FieldKindLabels: true,
	FieldKindDate: true, FieldKindCheckbox: true, FieldKindURL: true, FieldKindEmail: true,
	FieldKindPhone: true, FieldKindMoney: true, FieldKindProgress: true, FieldKindPeople: true,
}

func IsValidFieldKind(k string) bool { return validFieldKinds[k] }

type CustomField struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	ListID      *uuid.UUID      `json:"list_id,omitempty"`
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Config      json.RawMessage `json:"config"`
	Required    bool            `json:"required"`
	OrderIndex  int             `json:"order_index"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type CustomValue struct {
	TaskID  uuid.UUID       `json:"task_id"`
	FieldID uuid.UUID       `json:"field_id"`
	Value   json.RawMessage `json:"value"`
}

type CustomFieldRepo interface {
	Create(ctx context.Context, f *CustomField) error
	GetByID(ctx context.Context, id uuid.UUID) (*CustomField, error)
	ListByList(ctx context.Context, listID uuid.UUID) ([]CustomField, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]CustomField, error)
	Update(ctx context.Context, f *CustomField) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type CustomValueRepo interface {
	Upsert(ctx context.Context, v *CustomValue) error
	Delete(ctx context.Context, taskID, fieldID uuid.UUID) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]CustomValue, error)
}
