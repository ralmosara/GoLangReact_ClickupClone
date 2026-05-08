package list

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

type Service struct{ repo domain.ListRepo }

func New(repo domain.ListRepo) *Service { return &Service{repo: repo} }

type CreateInput struct {
	SpaceID  uuid.UUID  `json:"space_id"`
	FolderID *uuid.UUID `json:"folder_id,omitempty"`
	Name     string     `json:"name"`
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.List, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	l := &domain.List{SpaceID: in.SpaceID, FolderID: in.FolderID, Name: in.Name}
	if err := s.repo.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *Service) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]domain.List, error) {
	return s.repo.ListBySpace(ctx, spaceID)
}

func (s *Service) ListByFolder(ctx context.Context, folderID uuid.UUID) ([]domain.List, error) {
	return s.repo.ListByFolder(ctx, folderID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.List, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, l *domain.List) error {
	return s.repo.Update(ctx, l)
}

func (s *Service) Archive(ctx context.Context, id uuid.UUID) (*domain.List, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, errors.New("not found")
	}
	if l.Archived {
		return l, nil
	}
	if err := s.repo.SetArchived(ctx, id, true); err != nil {
		return nil, err
	}
	l.Archived = true
	return l, nil
}

func (s *Service) Unarchive(ctx context.Context, id uuid.UUID) (*domain.List, error) {
	l, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, errors.New("not found")
	}
	if !l.Archived {
		return l, nil
	}
	if err := s.repo.SetArchived(ctx, id, false); err != nil {
		return nil, err
	}
	l.Archived = false
	return l, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
