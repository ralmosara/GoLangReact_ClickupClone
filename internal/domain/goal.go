package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	TargetKindNumber    = "number"
	TargetKindCurrency  = "currency"
	TargetKindBoolean   = "boolean"
	TargetKindTaskDone  = "task_completed"
)

type Goal struct {
	ID           uuid.UUID  `json:"id"`
	WorkspaceID  uuid.UUID  `json:"workspace_id"`
	ParentGoalID *uuid.UUID `json:"parent_goal_id,omitempty"`
	OwnerID      *uuid.UUID `json:"owner_id,omitempty"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	StartAt      *time.Time `json:"start_at,omitempty"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	Archived     bool       `json:"archived"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Derived — not persisted. Populated by the service when listing.
	Progress float64 `json:"progress,omitempty"`
}

type GoalTarget struct {
	ID             uuid.UUID  `json:"id"`
	GoalID         uuid.UUID  `json:"goal_id"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind"`
	TargetNumber   *float64   `json:"target_number,omitempty"`
	CurrentNumber  float64    `json:"current_number"`
	Currency       *string    `json:"currency,omitempty"`
	TargetBoolean  *bool      `json:"target_boolean,omitempty"`
	CurrentBoolean bool       `json:"current_boolean"`
	TaskListID     *uuid.UUID `json:"task_list_id,omitempty"`
	OrderIndex     int        `json:"order_index"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Derived metrics — computed for `task_completed` targets.
	TotalTasks     *int    `json:"total_tasks,omitempty"`
	CompletedTasks *int    `json:"completed_tasks,omitempty"`
	Progress       float64 `json:"progress"`
}

type GoalRepo interface {
	Create(ctx context.Context, g *Goal) error
	GetByID(ctx context.Context, id uuid.UUID) (*Goal, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Goal, error)
	ListChildren(ctx context.Context, parentID uuid.UUID) ([]Goal, error)
	Update(ctx context.Context, g *Goal) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type GoalTargetRepo interface {
	Create(ctx context.Context, t *GoalTarget) error
	GetByID(ctx context.Context, id uuid.UUID) (*GoalTarget, error)
	ListByGoal(ctx context.Context, goalID uuid.UUID) ([]GoalTarget, error)
	Update(ctx context.Context, t *GoalTarget) error
	Delete(ctx context.Context, id uuid.UUID) error

	// TaskCompletion returns (completed, total) for the given list,
	// used by the service to hydrate `task_completed` target progress.
	TaskCompletion(ctx context.Context, listID uuid.UUID) (completed, total int, err error)
}
