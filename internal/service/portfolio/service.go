// Package portfolio is the service for cross-space rollups. CRUD plus
// the Rollup() helper that powers the executive dashboard widget.
package portfolio

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
)

type Service struct {
	repo  domain.PortfolioRepo
	audit *audit.Recorder
}

func New(repo domain.PortfolioRepo, recorder *audit.Recorder) *Service {
	return &Service{repo: repo, audit: recorder}
}

type CreateInput struct {
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Color       *string    `json:"color,omitempty"`
	OwnerID     *uuid.UUID `json:"owner_id,omitempty"`
}

type UpdateInput struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Color       *string    `json:"color,omitempty"`
	OwnerID     *uuid.UUID `json:"owner_id,omitempty"`
}

var (
	ErrNameRequired = errors.New("name is required")
	ErrNotFound     = errors.New("portfolio not found")
)

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Portfolio, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, ErrNameRequired
	}
	p := &domain.Portfolio{
		WorkspaceID: in.WorkspaceID,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		Color:       in.Color,
		OwnerID:     in.OwnerID,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.record(ctx, actor, p.ID, "created", nil, p)
	return p, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Portfolio, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Service) List(ctx context.Context, workspaceID uuid.UUID) ([]domain.Portfolio, error) {
	return s.repo.ListByWorkspace(ctx, workspaceID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Portfolio, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, ErrNameRequired
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
	existing.OwnerID = in.OwnerID
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	s.record(ctx, actor, id, "updated", &before, existing)
	return existing, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.record(ctx, actor, id, "deleted", nil, nil)
	return nil
}

func (s *Service) AttachSpace(ctx context.Context, actor, portfolioID, spaceID uuid.UUID) error {
	if err := s.repo.AttachSpace(ctx, portfolioID, spaceID); err != nil {
		return err
	}
	s.record(ctx, actor, portfolioID, "space_attached", nil, map[string]any{"space_id": spaceID})
	return nil
}

func (s *Service) DetachSpace(ctx context.Context, actor, portfolioID, spaceID uuid.UUID) error {
	if err := s.repo.DetachSpace(ctx, portfolioID, spaceID); err != nil {
		return err
	}
	s.record(ctx, actor, portfolioID, "space_detached", nil, map[string]any{"space_id": spaceID})
	return nil
}

func (s *Service) Rollup(ctx context.Context, portfolioID uuid.UUID) (*domain.PortfolioRollup, error) {
	return s.repo.Rollup(ctx, portfolioID)
}

func (s *Service) record(ctx context.Context, actor, id uuid.UUID, verb string, before, after any) {
	if s.audit == nil {
		return
	}
	entityID := id
	s.audit.Record(ctx, audit.Entry{
		ActorID:    &actor,
		EntityType: "portfolio",
		EntityID:   &entityID,
		Verb:       verb,
		Before:     before,
		After:      after,
	})
}
