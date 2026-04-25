package notification

import (
	"context"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

type Service struct {
	repo domain.NotificationRepo
}

func New(repo domain.NotificationRepo) *Service { return &Service{repo: repo} }

func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit int) ([]domain.Notification, error) {
	return s.repo.ListByUser(ctx, userID, unreadOnly, limit)
}

func (s *Service) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkRead(ctx, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}
