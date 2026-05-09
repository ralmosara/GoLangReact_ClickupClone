// Package milestone is the service layer for date-anchored deliverables.
// Distinct from sprints: a milestone is a single ship date that pulls
// tasks from anywhere in the workspace, not a time-boxed iteration.
package milestone

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
	repo  domain.MilestoneRepo
	audit *audit.Recorder
}

func New(repo domain.MilestoneRepo, recorder *audit.Recorder) *Service {
	return &Service{repo: repo, audit: recorder}
}

type CreateInput struct {
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Color       *string    `json:"color,omitempty"`
	DueAt       time.Time  `json:"due_at"`
	OwnerID     *uuid.UUID `json:"owner_id,omitempty"`
}

type UpdateInput struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Color       *string                 `json:"color,omitempty"`
	DueAt       time.Time               `json:"due_at"`
	Status      domain.MilestoneStatus  `json:"status"`
	OwnerID     *uuid.UUID              `json:"owner_id,omitempty"`
}

var (
	ErrNameRequired = errors.New("name is required")
	ErrDueRequired  = errors.New("due_at is required")
	ErrNotFound     = errors.New("milestone not found")
	ErrBadStatus    = errors.New("status must be planned, in_progress, shipped, or cancelled")
)

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Milestone, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, ErrNameRequired
	}
	if in.DueAt.IsZero() {
		return nil, ErrDueRequired
	}
	m := &domain.Milestone{
		WorkspaceID: in.WorkspaceID,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		Color:       in.Color,
		DueAt:       in.DueAt,
		Status:      domain.MilestonePlanned,
		OwnerID:     in.OwnerID,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	s.record(ctx, &actor, m.ID, "created", nil, m)
	return m, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Milestone, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrNotFound
	}
	return m, nil
}

func (s *Service) List(ctx context.Context, workspaceID uuid.UUID) ([]domain.Milestone, error) {
	return s.repo.ListByWorkspace(ctx, workspaceID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Milestone, error) {
	if !validStatus(in.Status) {
		return nil, ErrBadStatus
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, ErrNameRequired
	}
	if in.DueAt.IsZero() {
		return nil, ErrDueRequired
	}
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrNotFound
	}
	before := *existing
	existing.Name = strings.TrimSpace(in.Name)
	existing.Description = in.Description
	existing.Color = in.Color
	existing.DueAt = in.DueAt
	existing.Status = in.Status
	existing.OwnerID = in.OwnerID
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	s.record(ctx, &actor, id, "updated", &before, existing)
	return existing, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.record(ctx, &actor, id, "deleted", nil, nil)
	return nil
}

func (s *Service) AttachTask(ctx context.Context, actor, milestoneID, taskID uuid.UUID) error {
	if err := s.repo.AttachTask(ctx, milestoneID, taskID, actor); err != nil {
		return err
	}
	s.record(ctx, &actor, milestoneID, "task_attached", nil, map[string]any{"task_id": taskID})
	return nil
}

func (s *Service) DetachTask(ctx context.Context, actor, milestoneID, taskID uuid.UUID) error {
	if err := s.repo.DetachTask(ctx, milestoneID, taskID); err != nil {
		return err
	}
	s.record(ctx, &actor, milestoneID, "task_detached", nil, map[string]any{"task_id": taskID})
	return nil
}

func (s *Service) ListTasks(ctx context.Context, milestoneID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.ListTasks(ctx, milestoneID)
}

func (s *Service) record(ctx context.Context, actor *uuid.UUID, id uuid.UUID, verb string, before, after any) {
	if s.audit == nil {
		return
	}
	entityID := id
	s.audit.Record(ctx, audit.Entry{
		ActorID:    actor,
		EntityType: "milestone",
		EntityID:   &entityID,
		Verb:       verb,
		Before:     before,
		After:      after,
	})
}

func validStatus(s domain.MilestoneStatus) bool {
	switch s {
	case domain.MilestonePlanned, domain.MilestoneInProgress, domain.MilestoneShipped, domain.MilestoneCancelled:
		return true
	}
	return false
}
