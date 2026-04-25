package space

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

type Service struct{ repo domain.SpaceRepo }

func New(repo domain.SpaceRepo) *Service { return &Service{repo: repo} }

type CreateInput struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Color       *string   `json:"color,omitempty"`
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Space, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	sp := &domain.Space{WorkspaceID: in.WorkspaceID, Name: in.Name, Color: in.Color}
	if err := s.repo.Create(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

func (s *Service) List(ctx context.Context, workspaceID uuid.UUID) ([]domain.Space, error) {
	return s.repo.ListByWorkspace(ctx, workspaceID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Space, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, sp *domain.Space) error {
	return s.repo.Update(ctx, sp)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
