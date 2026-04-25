package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Doc struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	SpaceID     *uuid.UUID      `json:"space_id,omitempty"`
	ParentID    *uuid.UUID      `json:"parent_id,omitempty"`
	CreatorID   *uuid.UUID      `json:"creator_id,omitempty"`
	Title       string          `json:"title"`
	Icon        *string         `json:"icon,omitempty"`
	Content     json.RawMessage `json:"content"`
	ContentText string          `json:"content_text,omitempty"`
	Version     int             `json:"version"`
	Archived    bool            `json:"archived"`
	OrderIndex  int             `json:"order_index"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type DocRepo interface {
	Create(ctx context.Context, d *Doc) error
	GetByID(ctx context.Context, id uuid.UUID) (*Doc, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Doc, error)
	ListChildren(ctx context.Context, parentID uuid.UUID) ([]Doc, error)
	Update(ctx context.Context, d *Doc) error
	Delete(ctx context.Context, id uuid.UUID) error
}
