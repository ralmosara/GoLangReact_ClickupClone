package assignee

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Add(ctx context.Context, taskID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO task_assignees (task_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
		taskID, userID)
	return err
}

func (r *Repo) Remove(ctx context.Context, taskID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM task_assignees WHERE task_id=$1 AND user_id=$2`, taskID, userID)
	return err
}

func (r *Repo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT user_id FROM task_assignees WHERE task_id=$1 ORDER BY assigned_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *Repo) ListTasksByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT task_id FROM task_assignees WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
