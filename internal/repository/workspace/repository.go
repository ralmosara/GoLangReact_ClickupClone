package workspace

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

func (r *Repo) Create(ctx context.Context, w *domain.Workspace) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, w.Name, w.Slug, w.OwnerID).Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Workspace, error) {
	var w domain.Workspace
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, COALESCE(owner_id, '00000000-0000-0000-0000-000000000000'::uuid), created_at, updated_at
		FROM workspaces WHERE id = $1
	`, id).Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *Repo) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT w.id, w.name, w.slug, COALESCE(w.owner_id, '00000000-0000-0000-0000-000000000000'::uuid), w.created_at, w.updated_at
		FROM workspaces w
		JOIN workspace_members m ON m.workspace_id = w.id
		WHERE m.user_id = $1
		ORDER BY w.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Workspace
	for rows.Next() {
		var w domain.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repo) AddMember(ctx context.Context, m *domain.WorkspaceMember) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (workspace_id, user_id) DO NOTHING
	`, m.WorkspaceID, m.UserID, m.Role)
	return err
}

func (r *Repo) IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2)
	`, workspaceID, userID).Scan(&exists)
	return exists, err
}
