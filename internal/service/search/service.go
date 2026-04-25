package search

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

type Service struct {
	repo domain.SearchRepo
}

func New(repo domain.SearchRepo) *Service { return &Service{repo: repo} }

func (s *Service) Search(ctx context.Context, workspaceID uuid.UUID, query string, limit int) ([]domain.SearchHit, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []domain.SearchHit{}, nil
	}
	return s.repo.Search(ctx, workspaceID, q, limit)
}
