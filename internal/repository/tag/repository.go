package tag

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Create(ctx context.Context, t *domain.Tag) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO tags (workspace_id, name, color)
		VALUES ($1,$2,$3)
		ON CONFLICT (workspace_id, name) DO UPDATE SET color = EXCLUDED.color
		RETURNING id
	`, t.WorkspaceID, t.Name, t.Color).Scan(&t.ID)
}

func (r *Repo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, workspace_id, name, color FROM tags WHERE workspace_id=$1 ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) AttachToTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		taskID, tagID)
	return err
}

func (r *Repo) DetachFromTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM task_tags WHERE task_id=$1 AND tag_id=$2`, taskID, tagID)
	return err
}

func (r *Repo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.workspace_id, t.name, t.color
		FROM tags t JOIN task_tags tt ON tt.tag_id = t.id
		WHERE tt.task_id = $1 ORDER BY t.name
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetByID is a convenience used by tag handlers.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	var t domain.Tag
	err := r.pool.QueryRow(ctx, `SELECT id, workspace_id, name, color FROM tags WHERE id=$1`, id).
		Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Color)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
