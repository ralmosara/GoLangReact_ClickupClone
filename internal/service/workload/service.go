// Package workload aggregates current per-assignee task load over a time
// window. Backs the Workload / Capacity view that PMs use to balance
// who's overcommitted this sprint.
//
// The service is a thin shell over a TaskWorkload provider so the
// concrete repo doesn't need to land in the domain interface — the
// shape is specific to this view and would clutter domain.TaskRepo.
package workload

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
	taskRepo "github.com/yourorg/clickup/internal/repository/task"
)

// Lookups is the optional dependency block. Hydrate decorates each
// bucket with the assignee's display name + email so the FE doesn't
// need a follow-up query per row. Nil = leave names empty (rare for
// production, fine for tests).
type Lookups struct {
	GetUser func(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type Service struct {
	tasks   *taskRepo.Repo
	lookups Lookups
}

func New(tasks *taskRepo.Repo, lookups Lookups) *Service {
	return &Service{tasks: tasks, lookups: lookups}
}

// Bucket is one row in the workload table. Mirrors taskRepo.WorkloadBucket
// plus the user metadata so the wire shape is self-contained.
type Bucket struct {
	UserID            uuid.UUID `json:"user_id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	OpenTaskCount     int       `json:"open_task_count"`
	OpenEstimateSec   int       `json:"open_estimate_seconds"`
	CompletedTasks    int       `json:"completed_task_count"`
	CompletedEstSec   int       `json:"completed_estimate_seconds"`
	OverdueTasks      int       `json:"overdue_task_count"`
}

type Input struct {
	WorkspaceID uuid.UUID
	From        time.Time
	To          time.Time
}

var ErrInvalidWindow = errors.New("from must be before to")

func (s *Service) Report(ctx context.Context, in Input) ([]Bucket, error) {
	if !in.From.Before(in.To) {
		return nil, ErrInvalidWindow
	}
	rows, err := s.tasks.Workload(ctx, in.WorkspaceID, in.From, in.To)
	if err != nil {
		return nil, err
	}
	out := make([]Bucket, 0, len(rows))
	for _, r := range rows {
		b := Bucket{
			UserID:          r.UserID,
			OpenTaskCount:   r.OpenTaskCount,
			OpenEstimateSec: r.OpenEstimateSec,
			CompletedTasks:  r.CompletedTasks,
			CompletedEstSec: r.CompletedEstSec,
			OverdueTasks:    r.OverdueTasks,
		}
		if s.lookups.GetUser != nil && r.UserID != uuid.Nil {
			if u, _ := s.lookups.GetUser(ctx, r.UserID); u != nil {
				b.Name = u.Name
				b.Email = u.Email
			}
		}
		if b.Name == "" && r.UserID == uuid.Nil {
			b.Name = "Unassigned"
		}
		out = append(out, b)
	}
	return out, nil
}
