package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Automation triggers supported by M4.
const (
	TriggerStatusChanged = "task.status_changed"
	TriggerCreated       = "task.created"
	TriggerAssigned      = "task.assigned"
	TriggerCompleted     = "task.completed"
	TriggerDueSoon       = "task.due_soon" // cron-polled
)

// Automation action types.
const (
	ActionChangeStatus = "change_status"
	ActionAssignUser   = "assign_user"
	ActionAddTag       = "add_tag"
	ActionAddComment   = "add_comment"
	ActionNotify       = "notify"
)

type Automation struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	ListID      *uuid.UUID      `json:"list_id,omitempty"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Trigger     json.RawMessage `json:"trigger"`
	Conditions  json.RawMessage `json:"conditions"`
	Actions     json.RawMessage `json:"actions"`
	Enabled     bool            `json:"enabled"`
	LastRunAt   *time.Time      `json:"last_run_at,omitempty"`
	RunCount    int             `json:"run_count"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type AutomationRepo interface {
	Create(ctx context.Context, a *Automation) error
	GetByID(ctx context.Context, id uuid.UUID) (*Automation, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Automation, error)
	ListByList(ctx context.Context, listID uuid.UUID) ([]Automation, error)
	ListEnabledForTrigger(ctx context.Context, listID *uuid.UUID, workspaceID uuid.UUID, triggerType string) ([]Automation, error)
	Update(ctx context.Context, a *Automation) error
	Delete(ctx context.Context, id uuid.UUID) error

	// RecordRun bumps LastRunAt and RunCount after an automation fires.
	RecordRun(ctx context.Context, id uuid.UUID) error
}
