package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Status is a per-list custom status shown as a Kanban column.
type Status struct {
	ID         uuid.UUID `json:"id"`
	ListID     uuid.UUID `json:"list_id"`
	Name       string    `json:"name"`
	Color      string    `json:"color"`
	Category   string    `json:"category"` // active | done | closed (ClickUp-style)
	OrderIndex int       `json:"order_index"`
	CreatedAt  time.Time `json:"created_at"`
}

type StatusRepo interface {
	Create(ctx context.Context, s *Status) error
	GetByID(ctx context.Context, id uuid.UUID) (*Status, error)
	ListByList(ctx context.Context, listID uuid.UUID) ([]Status, error)
	Update(ctx context.Context, s *Status) error
	Delete(ctx context.Context, id uuid.UUID) error
}
