package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorkspaceMember struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

type WorkspaceRepo interface {
	Create(ctx context.Context, w *Workspace) error
	GetByID(ctx context.Context, id uuid.UUID) (*Workspace, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]Workspace, error)
	AddMember(ctx context.Context, m *WorkspaceMember) error
	IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)
}
