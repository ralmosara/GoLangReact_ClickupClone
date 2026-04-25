package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Space struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Color       *string   `json:"color,omitempty"`
	Position    int       `json:"position"`
	Archived    bool      `json:"archived"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SpaceRepo interface {
	Create(ctx context.Context, s *Space) error
	GetByID(ctx context.Context, id uuid.UUID) (*Space, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Space, error)
	Update(ctx context.Context, s *Space) error
	Delete(ctx context.Context, id uuid.UUID) error
}
