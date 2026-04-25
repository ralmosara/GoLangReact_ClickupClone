package timeentry

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
)

type Service struct {
	repo  domain.TimeEntryRepo
	audit *audit.Recorder
}

func New(repo domain.TimeEntryRepo, rec *audit.Recorder) *Service {
	return &Service{repo: repo, audit: rec}
}

type CreateInput struct {
	TaskID    uuid.UUID  `json:"task_id"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	DurationS *int       `json:"duration_s,omitempty"`
	Note      string     `json:"note,omitempty"`
	Billable  bool       `json:"billable,omitempty"`
}

type UpdateInput struct {
	StartedAt *time.Time `json:"started_at,omitempty"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	DurationS *int       `json:"duration_s,omitempty"`
	Note      *string    `json:"note,omitempty"`
	Billable  *bool      `json:"billable,omitempty"`
}

// StartTimer opens a new running timer for the actor. Any existing running
// timer on any task is stopped first (ClickUp behaviour — one timer at a time).
func (s *Service) StartTimer(ctx context.Context, actor, taskID uuid.UUID, note string) (*domain.TimeEntry, error) {
	if err := s.stopAllActive(ctx, actor); err != nil {
		return nil, err
	}
	e := &domain.TimeEntry{
		TaskID:    taskID,
		UserID:    actor,
		StartedAt: time.Now().UTC(),
		Note:      note,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "time_entry",
			EntityID:   &e.ID,
			Verb:       "started",
			After:      e,
		})
	}
	return e, nil
}

func (s *Service) stopAllActive(ctx context.Context, userID uuid.UUID) error {
	running, err := s.repo.ActiveForUser(ctx, userID)
	if err != nil {
		return err
	}
	for i := range running {
		e := &running[i]
		if err := s.finalize(e, time.Now().UTC()); err != nil {
			return err
		}
		if err := s.repo.Update(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

// StopTimer closes a running entry and computes its duration.
func (s *Service) StopTimer(ctx context.Context, actor, id uuid.UUID) (*domain.TimeEntry, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errors.New("not found")
	}
	if e.UserID != actor {
		return nil, errors.New("forbidden")
	}
	if e.StoppedAt != nil {
		return e, nil
	}
	if err := s.finalize(e, time.Now().UTC()); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "time_entry",
			EntityID:   &e.ID,
			Verb:       "stopped",
			After:      e,
		})
	}
	return e, nil
}

func (s *Service) finalize(e *domain.TimeEntry, stopped time.Time) error {
	if stopped.Before(e.StartedAt) {
		return errors.New("stopped_at before started_at")
	}
	e.StoppedAt = &stopped
	d := int(stopped.Sub(e.StartedAt).Seconds())
	e.DurationS = &d
	return nil
}

// CreateManual adds a past time entry with explicit start/stop.
func (s *Service) CreateManual(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.TimeEntry, error) {
	if in.StartedAt == nil {
		return nil, errors.New("started_at required")
	}
	e := &domain.TimeEntry{
		TaskID:    in.TaskID,
		UserID:    actor,
		StartedAt: *in.StartedAt,
		StoppedAt: in.StoppedAt,
		DurationS: in.DurationS,
		Note:      in.Note,
		Billable:  in.Billable,
	}
	if e.StoppedAt != nil && e.DurationS == nil {
		d := int(e.StoppedAt.Sub(e.StartedAt).Seconds())
		if d < 0 {
			return nil, errors.New("stopped_at before started_at")
		}
		e.DurationS = &d
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "time_entry",
			EntityID:   &e.ID,
			Verb:       "logged",
			After:      e,
		})
	}
	return e, nil
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.TimeEntry, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errors.New("not found")
	}
	if e.UserID != actor {
		return nil, errors.New("forbidden")
	}
	before := *e
	if in.StartedAt != nil {
		e.StartedAt = *in.StartedAt
	}
	if in.StoppedAt != nil {
		e.StoppedAt = in.StoppedAt
	}
	if in.DurationS != nil {
		e.DurationS = in.DurationS
	}
	if in.Note != nil {
		e.Note = *in.Note
	}
	if in.Billable != nil {
		e.Billable = *in.Billable
	}
	// Recompute duration when start/stop set without explicit duration.
	if e.StoppedAt != nil && in.DurationS == nil {
		d := int(e.StoppedAt.Sub(e.StartedAt).Seconds())
		e.DurationS = &d
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "time_entry",
			EntityID:   &e.ID,
			Verb:       "updated",
			Before:     before,
			After:      e,
		})
	}
	return e, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	e, _ := s.repo.GetByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil && e != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "time_entry",
			EntityID:   &e.ID,
			Verb:       "deleted",
			Before:     e,
		})
	}
	return nil
}

func (s *Service) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.TimeEntry, error) {
	return s.repo.ListByTask(ctx, taskID)
}

func (s *Service) ActiveForUser(ctx context.Context, userID uuid.UUID) ([]domain.TimeEntry, error) {
	return s.repo.ActiveForUser(ctx, userID)
}

func (s *Service) Report(ctx context.Context, f domain.TimeEntryFilter) ([]domain.TimeReportBucket, error) {
	return s.repo.Report(ctx, f)
}
