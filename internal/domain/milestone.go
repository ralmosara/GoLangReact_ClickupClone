package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MilestoneStatus is the lifecycle of a milestone. Closed milestones
// (shipped/cancelled) are preserved so historical roadmaps stay intact.
type MilestoneStatus string

const (
	MilestonePlanned    MilestoneStatus = "planned"
	MilestoneInProgress MilestoneStatus = "in_progress"
	MilestoneShipped    MilestoneStatus = "shipped"
	MilestoneCancelled  MilestoneStatus = "cancelled"
)

type Milestone struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Color       *string         `json:"color,omitempty"`
	DueAt       time.Time       `json:"due_at"`
	Status      MilestoneStatus `json:"status"`
	OwnerID     *uuid.UUID      `json:"owner_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`

	// Aggregates populated on list/get; nil on writes. The UI uses
	// these to render "12 of 30 tasks done · 40%" without a follow-up
	// query.
	TaskCount      int `json:"task_count"`
	CompletedTasks int `json:"completed_tasks"`
}

type MilestoneRepo interface {
	Create(ctx context.Context, m *Milestone) error
	Get(ctx context.Context, id uuid.UUID) (*Milestone, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Milestone, error)
	Update(ctx context.Context, m *Milestone) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Task linkage --------------------------------------------------------

	// AttachTask is idempotent — re-attaching is a no-op via ON CONFLICT.
	AttachTask(ctx context.Context, milestoneID, taskID, actor uuid.UUID) error
	DetachTask(ctx context.Context, milestoneID, taskID uuid.UUID) error
	// ListTasks returns the task IDs attached to the milestone, ordered
	// by added_at ASC. Caller hydrates the task rows themselves —
	// keeps this repo from needing a join into tasks.
	ListTasks(ctx context.Context, milestoneID uuid.UUID) ([]uuid.UUID, error)
}
