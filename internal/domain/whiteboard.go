package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Whiteboard struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	SpaceID     *uuid.UUID      `json:"space_id,omitempty"`
	Name        string          `json:"name"`
	Snapshot    json.RawMessage `json:"snapshot"`
	Version     int             `json:"version"`
	CreatorID   *uuid.UUID      `json:"creator_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type WhiteboardRepo interface {
	Create(ctx context.Context, w *Whiteboard) error
	GetByID(ctx context.Context, id uuid.UUID) (*Whiteboard, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Whiteboard, error)
	ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]Whiteboard, error)
	Update(ctx context.Context, w *Whiteboard) error
	Delete(ctx context.Context, id uuid.UUID) error
}
