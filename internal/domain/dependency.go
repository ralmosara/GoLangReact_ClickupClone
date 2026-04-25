package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Dependency kinds. `waiting_on` means task_id is waiting on depends_on_id
// (so depends_on_id must complete first). `blocks` is the inverse; we store
// only `waiting_on` rows and flip direction in the UI.
const (
	DepKindWaitingOn = "waiting_on"
	DepKindBlocks    = "blocks"
)

type TaskDependency struct {
	TaskID      uuid.UUID `json:"task_id"`
	DependsOnID uuid.UUID `json:"depends_on_id"`
	Kind        string    `json:"kind"`
	CreatedAt   time.Time `json:"created_at"`
}

type DependencyRepo interface {
	Create(ctx context.Context, d *TaskDependency) error
	Delete(ctx context.Context, taskID, dependsOnID uuid.UUID) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]TaskDependency, error)

	// ListDependents returns rows where depends_on_id = taskID (tasks that are
	// waiting on this one, i.e. this task is blocking them).
	ListDependents(ctx context.Context, taskID uuid.UUID) ([]TaskDependency, error)

	// WouldCreateCycle walks the dependency graph to detect whether adding
	// `taskID waits on dependsOnID` would close a cycle.
	WouldCreateCycle(ctx context.Context, taskID, dependsOnID uuid.UUID) (bool, error)
}
