package space

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

func (r *Repo) Create(ctx context.Context, s *domain.Space) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO spaces (workspace_id, name, color, position)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, s.WorkspaceID, s.Name, s.Color, s.Position).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Space, error) {
	var s domain.Space
	err := r.pool.QueryRow(ctx, `
		SELECT id, workspace_id, name, color, position, archived, created_at, updated_at
		FROM spaces WHERE id = $1
	`, id).Scan(&s.ID, &s.WorkspaceID, &s.Name, &s.Color, &s.Position, &s.Archived, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Space, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, workspace_id, name, color, position, archived, created_at, updated_at
		FROM spaces WHERE workspace_id = $1 AND archived = FALSE
		ORDER BY position, created_at
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Space
	for rows.Next() {
		var s domain.Space
		if err := rows.Scan(&s.ID, &s.WorkspaceID, &s.Name, &s.Color, &s.Position, &s.Archived, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, s *domain.Space) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE spaces SET name=$2, color=$3, position=$4, archived=$5, updated_at=NOW()
		WHERE id=$1
	`, s.ID, s.Name, s.Color, s.Position, s.Archived)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM spaces WHERE id=$1`, id)
	return err
}
