package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type List struct {
	ID        uuid.UUID  `json:"id"`
	SpaceID   uuid.UUID  `json:"space_id"`
	FolderID  *uuid.UUID `json:"folder_id,omitempty"`
	Name      string     `json:"name"`
	Position  int        `json:"position"`
	Archived  bool       `json:"archived"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ListRepo interface {
	Create(ctx context.Context, l *List) error
	GetByID(ctx context.Context, id uuid.UUID) (*List, error)
	ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]List, error)
	ListByFolder(ctx context.Context, folderID uuid.UUID) ([]List, error)
	Update(ctx context.Context, l *List) error
	SetArchived(ctx context.Context, id uuid.UUID, archived bool) error
	Delete(ctx context.Context, id uuid.UUID) error
}
