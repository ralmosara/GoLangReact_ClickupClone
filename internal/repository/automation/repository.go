package automation

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

const cols = `id, workspace_id, list_id, name, description, trigger, conditions, actions, enabled,
	last_run_at, run_count, created_at, updated_at`

func scan(row pgx.Row) (*domain.Automation, error) {
	var a domain.Automation
	err := row.Scan(&a.ID, &a.WorkspaceID, &a.ListID, &a.Name, &a.Description,
		&a.Trigger, &a.Conditions, &a.Actions, &a.Enabled,
		&a.LastRunAt, &a.RunCount, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func scanMany(rows pgx.Rows) ([]domain.Automation, error) {
	var out []domain.Automation
	for rows.Next() {
		var a domain.Automation
		if err := rows.Scan(&a.ID, &a.WorkspaceID, &a.ListID, &a.Name, &a.Description,
			&a.Trigger, &a.Conditions, &a.Actions, &a.Enabled,
			&a.LastRunAt, &a.RunCount, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, a *domain.Automation) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO automations (workspace_id, list_id, name, description, trigger, conditions, actions, enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at, updated_at
	`, a.WorkspaceID, a.ListID, a.Name, a.Description, a.Trigger, a.Conditions, a.Actions, a.Enabled).
		Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Automation, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM automations WHERE id=$1`, id))
}

func (r *Repo) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Automation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM automations WHERE workspace_id=$1 ORDER BY created_at`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.Automation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM automations WHERE list_id=$1 ORDER BY created_at`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

// ListEnabledForTrigger returns automations whose trigger.type matches and
// whose scope (workspace-level or list-level) covers the given listID.
func (r *Repo) ListEnabledForTrigger(ctx context.Context, listID *uuid.UUID, wsID uuid.UUID, triggerType string) ([]domain.Automation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+cols+` FROM automations
		WHERE enabled = TRUE
		  AND workspace_id = $1
		  AND (list_id IS NULL OR list_id = $2)
		  AND trigger ->> 'type' = $3
		ORDER BY created_at
	`, wsID, listID, triggerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) Update(ctx context.Context, a *domain.Automation) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE automations SET
			name=$2, description=$3, trigger=$4, conditions=$5, actions=$6, enabled=$7,
			list_id=$8, updated_at=NOW()
		WHERE id=$1
	`, a.ID, a.Name, a.Description, a.Trigger, a.Conditions, a.Actions, a.Enabled, a.ListID)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM automations WHERE id=$1`, id)
	return err
}

func (r *Repo) RecordRun(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE automations SET last_run_at = NOW(), run_count = run_count + 1 WHERE id=$1`, id)
	return err
}
