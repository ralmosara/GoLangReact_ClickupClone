package notification

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const cols = `id, user_id, actor_id, kind, entity_type, entity_id, payload, read_at, created_at`

func (r *Repo) Create(ctx context.Context, n *domain.Notification) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, actor_id, kind, entity_type, entity_id, payload)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at
	`, n.UserID, n.ActorID, n.Kind, n.EntityType, n.EntityID, n.Payload).Scan(&n.ID, &n.CreatedAt)
}

func (r *Repo) ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit int) ([]domain.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT ` + cols + ` FROM notifications WHERE user_id=$1`
	args := []any{userID}
	if unreadOnly {
		q += ` AND read_at IS NULL`
	}
	q += ` ORDER BY created_at DESC LIMIT $2`
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorID, &n.Kind, &n.EntityType, &n.EntityID, &n.Payload, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var c int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND read_at IS NULL`, userID).Scan(&c)
	return c, err
}

func (r *Repo) MarkRead(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE id=$1 AND read_at IS NULL`, id)
	return err
}

func (r *Repo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id=$1 AND read_at IS NULL`, userID)
	return err
}
