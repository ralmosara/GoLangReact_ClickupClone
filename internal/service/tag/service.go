package tag

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
)

type Service struct {
	repo  domain.TagRepo
	audit *audit.Recorder
}

func New(repo domain.TagRepo, rec *audit.Recorder) *Service {
	return &Service{repo: repo, audit: rec}
}

type CreateInput struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Color       *string   `json:"color,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Tag, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	t := &domain.Tag{WorkspaceID: in.WorkspaceID, Name: strings.TrimSpace(in.Name), Color: in.Color}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &in.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "tag",
			EntityID:    &t.ID,
			Verb:        "created",
			After:       t,
		})
	}
	return t, nil
}

func (s *Service) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Tag, error) {
	return s.repo.ListByWorkspace(ctx, workspaceID)
}

func (s *Service) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.Tag, error) {
	return s.repo.ListByTask(ctx, taskID)
}

func (s *Service) AttachToTask(ctx context.Context, actor, taskID, tagID uuid.UUID) error {
	if err := s.repo.AttachToTask(ctx, taskID, tagID); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "task",
			EntityID:   &taskID,
			Verb:       "tag.attached",
			After:      map[string]uuid.UUID{"tag_id": tagID},
		})
	}
	return nil
}

func (s *Service) DetachFromTask(ctx context.Context, actor, taskID, tagID uuid.UUID) error {
	if err := s.repo.DetachFromTask(ctx, taskID, tagID); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "task",
			EntityID:   &taskID,
			Verb:       "tag.detached",
			Before:     map[string]uuid.UUID{"tag_id": tagID},
		})
	}
	return nil
}
