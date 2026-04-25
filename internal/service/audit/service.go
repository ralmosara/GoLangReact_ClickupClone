package audit

import (
	"context"

	"github.com/yourorg/clickup/internal/domain"
)

type Service struct {
	repo domain.AuditRepo
}

func New(repo domain.AuditRepo) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f domain.AuditFilter) ([]domain.AuditEntry, error) {
	return s.repo.List(ctx, f)
}
