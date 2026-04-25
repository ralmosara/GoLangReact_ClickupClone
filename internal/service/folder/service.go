package folder

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

type Service struct{ repo domain.FolderRepo }

func New(repo domain.FolderRepo) *Service { return &Service{repo: repo} }

type CreateInput struct {
	SpaceID uuid.UUID `json:"space_id"`
	Name    string    `json:"name"`
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Folder, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	f := &domain.Folder{SpaceID: in.SpaceID, Name: in.Name}
	if err := s.repo.Create(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Service) List(ctx context.Context, spaceID uuid.UUID) ([]domain.Folder, error) {
	return s.repo.ListBySpace(ctx, spaceID)
}

func (s *Service) Update(ctx context.Context, f *domain.Folder) error {
	return s.repo.Update(ctx, f)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
