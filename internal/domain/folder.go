package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID        uuid.UUID `json:"id"`
	SpaceID   uuid.UUID `json:"space_id"`
	Name      string    `json:"name"`
	Position  int       `json:"position"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FolderRepo interface {
	Create(ctx context.Context, f *Folder) error
	GetByID(ctx context.Context, id uuid.UUID) (*Folder, error)
	ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]Folder, error)
	Update(ctx context.Context, f *Folder) error
	Delete(ctx context.Context, id uuid.UUID) error
}
