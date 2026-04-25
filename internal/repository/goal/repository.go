package goal

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

const cols = `id, workspace_id, parent_goal_id, owner_id, name, description, start_at, due_at, archived, created_at, updated_at`

func scan(row pgx.Row) (*domain.Goal, error) {
	var g domain.Goal
	err := row.Scan(&g.ID, &g.WorkspaceID, &g.ParentGoalID, &g.OwnerID, &g.Name, &g.Description,
		&g.StartAt, &g.DueAt, &g.Archived, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func scanMany(rows pgx.Rows) ([]domain.Goal, error) {
	var out []domain.Goal
	for rows.Next() {
		var g domain.Goal
		if err := rows.Scan(&g.ID, &g.WorkspaceID, &g.ParentGoalID, &g.OwnerID, &g.Name, &g.Description,
			&g.StartAt, &g.DueAt, &g.Archived, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, g *domain.Goal) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO goals (workspace_id, parent_goal_id, owner_id, name, description, start_at, due_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at
	`, g.WorkspaceID, g.ParentGoalID, g.OwnerID, g.Name, g.Description, g.StartAt, g.DueAt).
		Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Goal, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM goals WHERE id=$1`, id))
}

func (r *Repo) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Goal, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM goals WHERE workspace_id=$1 AND archived = FALSE ORDER BY created_at`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) ListChildren(ctx context.Context, parentID uuid.UUID) ([]domain.Goal, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM goals WHERE parent_goal_id=$1 AND archived = FALSE ORDER BY created_at`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) Update(ctx context.Context, g *domain.Goal) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE goals SET
			parent_goal_id=$2, owner_id=$3, name=$4, description=$5,
			start_at=$6, due_at=$7, archived=$8, updated_at=NOW()
		WHERE id=$1
	`, g.ID, g.ParentGoalID, g.OwnerID, g.Name, g.Description, g.StartAt, g.DueAt, g.Archived)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM goals WHERE id=$1`, id)
	return err
}
