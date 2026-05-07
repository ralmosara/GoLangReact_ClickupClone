package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        uuid.UUID  `json:"id"`
	TaskID    uuid.UUID  `json:"task_id"`
	AuthorID  *uuid.UUID `json:"author_id,omitempty"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CommentRepo interface {
	Create(ctx context.Context, c *Comment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Comment, error)
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]Comment, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
