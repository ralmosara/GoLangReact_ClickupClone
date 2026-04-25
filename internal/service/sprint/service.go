package sprint

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
)

type Service struct {
	repo  domain.SprintRepo
	audit *audit.Recorder
}

func New(repo domain.SprintRepo, rec *audit.Recorder) *Service {
	return &Service{repo: repo, audit: rec}
}

type CreateInput struct {
	ListID     uuid.UUID `json:"list_id"`
	Name       string    `json:"name"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	GoalPoints int       `json:"goal_points,omitempty"`
	Status     string    `json:"status,omitempty"`
}

type UpdateInput struct {
	Name       *string    `json:"name,omitempty"`
	StartsAt   *time.Time `json:"starts_at,omitempty"`
	EndsAt     *time.Time `json:"ends_at,omitempty"`
	GoalPoints *int       `json:"goal_points,omitempty"`
	Status     *string    `json:"status,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Sprint, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	if in.EndsAt.Before(in.StartsAt) {
		return nil, errors.New("ends_at must be after starts_at")
	}
	sp := &domain.Sprint{
		ListID:     in.ListID,
		Name:       strings.TrimSpace(in.Name),
		StartsAt:   in.StartsAt,
		EndsAt:     in.EndsAt,
		GoalPoints: in.GoalPoints,
		Status:     in.Status,
	}
	if err := s.repo.Create(ctx, sp); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "sprint",
			EntityID:   &sp.ID,
			Verb:       "created",
			After:      sp,
		})
	}
	return sp, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.Sprint, error) {
	return s.repo.ListByList(ctx, listID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Sprint, error) {
	sp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, errors.New("not found")
	}
	before := *sp
	if in.Name != nil {
		sp.Name = *in.Name
	}
	if in.StartsAt != nil {
		sp.StartsAt = *in.StartsAt
	}
	if in.EndsAt != nil {
		sp.EndsAt = *in.EndsAt
	}
	if in.GoalPoints != nil {
		sp.GoalPoints = *in.GoalPoints
	}
	if in.Status != nil {
		sp.Status = *in.Status
	}
	if sp.EndsAt.Before(sp.StartsAt) {
		return nil, errors.New("ends_at must be after starts_at")
	}
	if err := s.repo.Update(ctx, sp); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "sprint",
			EntityID:   &sp.ID,
			Verb:       "updated",
			Before:     before,
			After:      sp,
		})
	}
	return sp, nil
}

// CloseResult reports what happened when a sprint was closed.
type CloseResult struct {
	Closed       *domain.Sprint `json:"closed"`
	RolledOver   int            `json:"rolled_over"`
	NextSprintID *uuid.UUID     `json:"next_sprint_id,omitempty"`
}

// Close flips a sprint to 'closed' and auto-rolls its incomplete tasks into the
// next open sprint on the same list. If no future sprint exists, tasks are
// detached (sprint_id = null) so the team can assign them manually.
func (s *Service) Close(ctx context.Context, actor, id uuid.UUID) (*CloseResult, error) {
	sp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, errors.New("not found")
	}
	if sp.Status == domain.SprintStatusClosed {
		return &CloseResult{Closed: sp}, nil
	}
	sp.Status = domain.SprintStatusClosed
	if err := s.repo.Update(ctx, sp); err != nil {
		return nil, err
	}
	result := &CloseResult{Closed: sp}

	next, err := s.repo.NextOpenSprint(ctx, sp.ListID, sp.EndsAt)
	if err != nil {
		return result, err
	}
	var moveTo uuid.UUID
	if next != nil {
		moveTo = next.ID
		result.NextSprintID = &next.ID
	}
	if moveTo == uuid.Nil {
		// Detach tasks from this sprint — pass nil via the zero uuid interpreted by the repo.
		moveTo = uuid.Nil
	}
	moved, err := s.repo.MoveOpenTasks(ctx, sp.ID, moveTo)
	if err != nil {
		return result, err
	}
	result.RolledOver = moved

	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "sprint",
			EntityID:   &sp.ID,
			Verb:       "closed",
			After:      result,
		})
	}
	return result, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) Burndown(ctx context.Context, sprintID uuid.UUID) ([]domain.BurndownPoint, error) {
	return s.repo.BurndownSeries(ctx, sprintID)
}
