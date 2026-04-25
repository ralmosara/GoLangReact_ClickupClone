package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TaskAssignee is a single (task, user) association.
type TaskAssignee struct {
	TaskID     uuid.UUID `json:"task_id"`
	UserID     uuid.UUID `json:"user_id"`
	AssignedAt time.Time `json:"assigned_at"`
}

type AssigneeRepo interface {
	Add(ctx context.Context, taskID, userID uuid.UUID) error
	Remove(ctx context.Context, taskID, userID uuid.UUID) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]uuid.UUID, error)
	ListTasksByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
