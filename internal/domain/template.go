package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TemplateKindList = "list"
	TemplateKindDoc  = "doc"
	TemplateKindTask = "task"
)

type Template struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Snapshot    json.RawMessage `json:"snapshot"`
	CreatorID   *uuid.UUID      `json:"creator_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type TemplateRepo interface {
	Create(ctx context.Context, t *Template) error
	GetByID(ctx context.Context, id uuid.UUID) (*Template, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, kind string) ([]Template, error)
	Update(ctx context.Context, t *Template) error
	Delete(ctx context.Context, id uuid.UUID) error
}
