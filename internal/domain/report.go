package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AccomplishmentTask is a slim projection of a completed task for the
// accomplishments report. Includes the parent list's name so the UI can show
// context without a second fetch.
type AccomplishmentTask struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ListID      uuid.UUID `json:"list_id"`
	ListName    string    `json:"list_name"`
	Archived    bool      `json:"archived"`
	CompletedAt time.Time `json:"completed_at"`
}

// AccomplishmentBucket groups completed tasks by day or ISO week.
type AccomplishmentBucket struct {
	Period   string               `json:"period"`
	StartsAt time.Time            `json:"starts_at"`
	EndsAt   time.Time            `json:"ends_at"`
	Count    int                  `json:"count"`
	Tasks    []AccomplishmentTask `json:"tasks"`
}

type AccomplishmentFilter struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	From        time.Time
	To          time.Time
}

type ReportRepo interface {
	AccomplishedTasks(ctx context.Context, f AccomplishmentFilter) ([]AccomplishmentTask, error)
}
