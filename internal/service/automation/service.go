package automation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
)

type Service struct {
	repo  domain.AutomationRepo
	audit *audit.Recorder
}

func New(repo domain.AutomationRepo, rec *audit.Recorder) *Service {
	return &Service{repo: repo, audit: rec}
}

type CreateInput struct {
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	ListID      *uuid.UUID      `json:"list_id,omitempty"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Trigger     json.RawMessage `json:"trigger"`
	Conditions  json.RawMessage `json:"conditions,omitempty"`
	Actions     json.RawMessage `json:"actions"`
	Enabled     *bool           `json:"enabled,omitempty"`
}

type UpdateInput struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Trigger     json.RawMessage `json:"trigger,omitempty"`
	Conditions  json.RawMessage `json:"conditions,omitempty"`
	Actions     json.RawMessage `json:"actions,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
	ListID      *uuid.UUID      `json:"list_id,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Automation, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	if len(in.Trigger) == 0 {
		return nil, errors.New("trigger required")
	}
	if len(in.Actions) == 0 {
		return nil, errors.New("actions required")
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	conditions := in.Conditions
	if len(conditions) == 0 {
		conditions = []byte("[]")
	}
	a := &domain.Automation{
		WorkspaceID: in.WorkspaceID,
		ListID:      in.ListID,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		Trigger:     in.Trigger,
		Conditions:  conditions,
		Actions:     in.Actions,
		Enabled:     enabled,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &in.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "automation",
			EntityID:    &a.ID,
			Verb:        "created",
			After:       a,
		})
	}
	return a, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Automation, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Automation, error) {
	return s.repo.ListByWorkspace(ctx, wsID)
}

func (s *Service) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.Automation, error) {
	return s.repo.ListByList(ctx, listID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Automation, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errors.New("not found")
	}
	before := *a
	if in.Name != nil {
		a.Name = *in.Name
	}
	if in.Description != nil {
		a.Description = *in.Description
	}
	if len(in.Trigger) > 0 {
		a.Trigger = in.Trigger
	}
	if len(in.Conditions) > 0 {
		a.Conditions = in.Conditions
	}
	if len(in.Actions) > 0 {
		a.Actions = in.Actions
	}
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
	if in.ListID != nil {
		a.ListID = in.ListID
	}
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &a.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "automation",
			EntityID:    &a.ID,
			Verb:        "updated",
			Before:      before,
			After:       a,
		})
	}
	return a, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	a, _ := s.repo.GetByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil && a != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &a.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "automation",
			EntityID:    &a.ID,
			Verb:        "deleted",
			Before:      a,
		})
	}
	return nil
}
