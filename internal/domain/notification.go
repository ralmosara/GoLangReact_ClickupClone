package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID         uuid.UUID       `json:"id"`
	UserID     uuid.UUID       `json:"user_id"`
	ActorID    *uuid.UUID      `json:"actor_id,omitempty"`
	Kind       string          `json:"kind"`
	EntityType string          `json:"entity_type,omitempty"`
	EntityID   *uuid.UUID      `json:"entity_id,omitempty"`
	Payload    json.RawMessage `json:"payload"`
	ReadAt     *time.Time      `json:"read_at,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type NotificationRepo interface {
	Create(ctx context.Context, n *Notification) error
	ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit int) ([]Notification, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, id uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
}
