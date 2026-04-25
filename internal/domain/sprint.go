package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	SprintStatusPlanned = "planned"
	SprintStatusActive  = "active"
	SprintStatusClosed  = "closed"
)

type Sprint struct {
	ID         uuid.UUID `json:"id"`
	ListID     uuid.UUID `json:"list_id"`
	Name       string    `json:"name"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	GoalPoints int       `json:"goal_points"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SprintRepo interface {
	Create(ctx context.Context, s *Sprint) error
	GetByID(ctx context.Context, id uuid.UUID) (*Sprint, error)
	ListByList(ctx context.Context, listID uuid.UUID) ([]Sprint, error)
	Update(ctx context.Context, s *Sprint) error
	Delete(ctx context.Context, id uuid.UUID) error

	// MoveOpenTasks reassigns open (non-completed) tasks from one sprint to
	// another. Returns the count moved — used by auto-rollover.
	MoveOpenTasks(ctx context.Context, fromSprintID, toSprintID uuid.UUID) (int, error)

	// NextOpenSprint finds the earliest non-closed sprint on the same list
	// whose start is on/after `after`. Returns nil if none exists.
	NextOpenSprint(ctx context.Context, listID uuid.UUID, after time.Time) (*Sprint, error)

	// BurndownSeries returns one row per day in [starts_at, ends_at] with the
	// number of completed points and total points in the sprint at that day.
	BurndownSeries(ctx context.Context, sprintID uuid.UUID) ([]BurndownPoint, error)
}

type BurndownPoint struct {
	Day            time.Time `json:"day"`
	RemainingPoints int       `json:"remaining_points"`
	CompletedPoints int       `json:"completed_points"`
	TotalPoints     int       `json:"total_points"`
}