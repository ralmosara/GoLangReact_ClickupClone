package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TimeEntry struct {
	ID        uuid.UUID  `json:"id"`
	TaskID    uuid.UUID  `json:"task_id"`
	UserID    uuid.UUID  `json:"user_id"`
	StartedAt time.Time  `json:"started_at"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	DurationS *int       `json:"duration_s,omitempty"`
	Note      string     `json:"note"`
	Billable  bool       `json:"billable"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TimeReportBucket aggregates durations by user inside a time window.
type TimeReportBucket struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email,omitempty"`
	Name      string    `json:"name,omitempty"`
	TaskCount int       `json:"task_count"`
	TotalS    int64     `json:"total_s"`
	BillableS int64     `json:"billable_s"`
}

type TimeEntryFilter struct {
	WorkspaceID *uuid.UUID
	TaskID      *uuid.UUID
	UserID      *uuid.UUID
	From        *time.Time
	To          *time.Time
}

type TimeEntryRepo interface {
	Create(ctx context.Context, e *TimeEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*TimeEntry, error)
	Update(ctx context.Context, e *TimeEntry) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]TimeEntry, error)
	ActiveForUser(ctx context.Context, userID uuid.UUID) ([]TimeEntry, error)

	// Report aggregates entries for a workspace window grouped by user.
	Report(ctx context.Context, f TimeEntryFilter) ([]TimeReportBucket, error)
}
