package goal

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
	goals   domain.GoalRepo
	targets domain.GoalTargetRepo
	audit   *audit.Recorder
}

func New(goals domain.GoalRepo, targets domain.GoalTargetRepo, rec *audit.Recorder) *Service {
	return &Service{goals: goals, targets: targets, audit: rec}
}

type CreateGoalInput struct {
	WorkspaceID  uuid.UUID  `json:"workspace_id"`
	ParentGoalID *uuid.UUID `json:"parent_goal_id,omitempty"`
	OwnerID      *uuid.UUID `json:"owner_id,omitempty"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	StartAt      *time.Time `json:"start_at,omitempty"`
	DueAt        *time.Time `json:"due_at,omitempty"`
}

type UpdateGoalInput struct {
	Name         *string    `json:"name,omitempty"`
	Description  *string    `json:"description,omitempty"`
	OwnerID      *uuid.UUID `json:"owner_id,omitempty"`
	ParentGoalID *uuid.UUID `json:"parent_goal_id,omitempty"`
	StartAt      *time.Time `json:"start_at,omitempty"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	Archived     *bool      `json:"archived,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateGoalInput) (*domain.Goal, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	g := &domain.Goal{
		WorkspaceID:  in.WorkspaceID,
		ParentGoalID: in.ParentGoalID,
		OwnerID:      in.OwnerID,
		Name:         strings.TrimSpace(in.Name),
		Description:  in.Description,
		StartAt:      in.StartAt,
		DueAt:        in.DueAt,
	}
	if err := s.goals.Create(ctx, g); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &in.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "goal",
			EntityID:    &g.ID,
			Verb:        "created",
			After:       g,
		})
	}
	return g, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Goal, error) {
	g, err := s.goals.GetByID(ctx, id)
	if err != nil || g == nil {
		return g, err
	}
	g.Progress = s.computeGoalProgress(ctx, g.ID)
	return g, nil
}

// ListByWorkspace returns every goal with its progress hydrated.
func (s *Service) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Goal, error) {
	goals, err := s.goals.ListByWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	// Hydrate own-target progress; we skip descendant roll-up for M6 to keep the
	// cost linear in len(goals). Roll-up can be a DB view in M8.
	for i := range goals {
		goals[i].Progress = s.computeGoalProgress(ctx, goals[i].ID)
	}
	return goals, nil
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateGoalInput) (*domain.Goal, error) {
	g, err := s.goals.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, errors.New("not found")
	}
	before := *g
	if in.Name != nil {
		g.Name = *in.Name
	}
	if in.Description != nil {
		g.Description = *in.Description
	}
	if in.OwnerID != nil {
		g.OwnerID = in.OwnerID
	}
	if in.ParentGoalID != nil {
		g.ParentGoalID = in.ParentGoalID
	}
	if in.StartAt != nil {
		g.StartAt = in.StartAt
	}
	if in.DueAt != nil {
		g.DueAt = in.DueAt
	}
	if in.Archived != nil {
		g.Archived = *in.Archived
	}
	if err := s.goals.Update(ctx, g); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &g.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "goal",
			EntityID:    &g.ID,
			Verb:        "updated",
			Before:      before,
			After:       g,
		})
	}
	return g, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	g, _ := s.goals.GetByID(ctx, id)
	if err := s.goals.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil && g != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &g.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "goal",
			EntityID:    &g.ID,
			Verb:        "deleted",
			Before:      g,
		})
	}
	return nil
}

/* ------ targets ---------------------------------------------------------- */

type CreateTargetInput struct {
	Name          string     `json:"name"`
	Kind          string     `json:"kind"`
	TargetNumber  *float64   `json:"target_number,omitempty"`
	CurrentNumber float64    `json:"current_number,omitempty"`
	Currency      *string    `json:"currency,omitempty"`
	TargetBoolean *bool      `json:"target_boolean,omitempty"`
	TaskListID    *uuid.UUID `json:"task_list_id,omitempty"`
	OrderIndex    int        `json:"order_index,omitempty"`
}

type UpdateTargetInput struct {
	Name           *string    `json:"name,omitempty"`
	TargetNumber   *float64   `json:"target_number,omitempty"`
	CurrentNumber  *float64   `json:"current_number,omitempty"`
	Currency       *string    `json:"currency,omitempty"`
	TargetBoolean  *bool      `json:"target_boolean,omitempty"`
	CurrentBoolean *bool      `json:"current_boolean,omitempty"`
	TaskListID     *uuid.UUID `json:"task_list_id,omitempty"`
	OrderIndex     *int       `json:"order_index,omitempty"`
}

var validKinds = map[string]bool{
	domain.TargetKindNumber:   true,
	domain.TargetKindCurrency: true,
	domain.TargetKindBoolean:  true,
	domain.TargetKindTaskDone: true,
}

func (s *Service) AddTarget(ctx context.Context, actor, goalID uuid.UUID, in CreateTargetInput) (*domain.GoalTarget, error) {
	if !validKinds[in.Kind] {
		return nil, errors.New("invalid target kind")
	}
	if in.Kind == domain.TargetKindTaskDone && in.TaskListID == nil {
		return nil, errors.New("task_list_id required for task_completed target")
	}
	t := &domain.GoalTarget{
		GoalID:        goalID,
		Name:          in.Name,
		Kind:          in.Kind,
		TargetNumber:  in.TargetNumber,
		CurrentNumber: in.CurrentNumber,
		Currency:      in.Currency,
		TargetBoolean: in.TargetBoolean,
		TaskListID:    in.TaskListID,
		OrderIndex:    in.OrderIndex,
	}
	if err := s.targets.Create(ctx, t); err != nil {
		return nil, err
	}
	s.hydrateTarget(ctx, t)
	return t, nil
}

func (s *Service) ListTargets(ctx context.Context, goalID uuid.UUID) ([]domain.GoalTarget, error) {
	targets, err := s.targets.ListByGoal(ctx, goalID)
	if err != nil {
		return nil, err
	}
	for i := range targets {
		s.hydrateTarget(ctx, &targets[i])
	}
	return targets, nil
}

func (s *Service) UpdateTarget(ctx context.Context, actor, id uuid.UUID, in UpdateTargetInput) (*domain.GoalTarget, error) {
	t, err := s.targets.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("not found")
	}
	if in.Name != nil {
		t.Name = *in.Name
	}
	if in.TargetNumber != nil {
		t.TargetNumber = in.TargetNumber
	}
	if in.CurrentNumber != nil {
		t.CurrentNumber = *in.CurrentNumber
	}
	if in.Currency != nil {
		t.Currency = in.Currency
	}
	if in.TargetBoolean != nil {
		t.TargetBoolean = in.TargetBoolean
	}
	if in.CurrentBoolean != nil {
		t.CurrentBoolean = *in.CurrentBoolean
	}
	if in.TaskListID != nil {
		t.TaskListID = in.TaskListID
	}
	if in.OrderIndex != nil {
		t.OrderIndex = *in.OrderIndex
	}
	if err := s.targets.Update(ctx, t); err != nil {
		return nil, err
	}
	s.hydrateTarget(ctx, t)
	return t, nil
}

func (s *Service) DeleteTarget(ctx context.Context, actor, id uuid.UUID) error {
	return s.targets.Delete(ctx, id)
}

/* ------ helpers ---------------------------------------------------------- */

// hydrateTarget fills the derived Progress / TotalTasks / CompletedTasks fields.
func (s *Service) hydrateTarget(ctx context.Context, t *domain.GoalTarget) {
	switch t.Kind {
	case domain.TargetKindNumber, domain.TargetKindCurrency:
		if t.TargetNumber != nil && *t.TargetNumber > 0 {
			t.Progress = clamp01(t.CurrentNumber / *t.TargetNumber)
		}
	case domain.TargetKindBoolean:
		if t.CurrentBoolean {
			t.Progress = 1
		}
	case domain.TargetKindTaskDone:
		if t.TaskListID != nil {
			done, total, err := s.targets.TaskCompletion(ctx, *t.TaskListID)
			if err == nil {
				t.CompletedTasks = &done
				t.TotalTasks = &total
				if total > 0 {
					t.Progress = clamp01(float64(done) / float64(total))
				}
			}
		}
	}
}

func (s *Service) computeGoalProgress(ctx context.Context, goalID uuid.UUID) float64 {
	targets, err := s.targets.ListByGoal(ctx, goalID)
	if err != nil || len(targets) == 0 {
		return 0
	}
	total := 0.0
	for i := range targets {
		s.hydrateTarget(ctx, &targets[i])
		total += targets[i].Progress
	}
	return clamp01(total / float64(len(targets)))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
