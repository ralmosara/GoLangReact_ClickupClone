package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID                uuid.UUID  `json:"id"`
	ListID            uuid.UUID  `json:"list_id"`
	ParentTaskID      *uuid.UUID `json:"parent_task_id,omitempty"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	Status            string     `json:"status"`
	StatusID          *uuid.UUID `json:"status_id,omitempty"`
	Priority          int        `json:"priority"`
	Position          float64    `json:"position"`
	AssigneeID        *uuid.UUID `json:"assignee_id,omitempty"`
	CreatorID         *uuid.UUID `json:"creator_id,omitempty"`
	DueAt             *time.Time `json:"due_at,omitempty"`
	StartAt           *time.Time `json:"start_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	Archived          bool       `json:"archived"`
	RecurringRule     *string    `json:"recurring_rule,omitempty"`
	RecurringParentID *uuid.UUID `json:"recurring_parent_id,omitempty"`
	SprintID          *uuid.UUID `json:"sprint_id,omitempty"`
	Points            *int       `json:"points,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type TaskFilter struct {
	ListID       *uuid.UUID
	AssigneeID   *uuid.UUID
	Status       *string
	StatusID     *uuid.UUID
	ParentTaskID *uuid.UUID
	IncludeSubs  bool
	Archived     *bool
}

type TaskRepo interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*Task, error)
	List(ctx context.Context, f TaskFilter) ([]Task, error)
	ListSubtasks(ctx context.Context, parentID uuid.UUID) ([]Task, error)
	Update(ctx context.Context, t *Task) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Reorder persists a new (status, status_id, position) tuple for drag-drop.
	// Callers compute position via gap-insertion (prev+next)/2.
	Reorder(ctx context.Context, id uuid.UUID, status string, statusID *uuid.UUID, position float64) error

	// MaxPosition returns the highest position in the given (list, status) bucket.
	MaxPosition(ctx context.Context, listID uuid.UUID, statusID *uuid.UUID) (float64, error)

	// SetArchived flips the archived flag without touching other columns.
	SetArchived(ctx context.Context, id uuid.UUID, archived bool) error
}
