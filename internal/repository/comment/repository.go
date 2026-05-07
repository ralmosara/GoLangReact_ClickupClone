package comment

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Create(ctx context.Context, c *domain.Comment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO comments (task_id, author_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, c.TaskID, c.AuthorID, c.Body).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	var c domain.Comment
	err := r.pool.QueryRow(ctx, `
		SELECT id, task_id, author_id, body, created_at, updated_at
		FROM comments WHERE id=$1
	`, id).Scan(&c.ID, &c.TaskID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, author_id, body, created_at, updated_at
		FROM comments WHERE task_id=$1
		ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM comments WHERE id=$1`, id)
	return err
}
