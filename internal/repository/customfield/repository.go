package customfield

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

const cols = `id, workspace_id, list_id, name, kind, config, required, order_index, created_at, updated_at`

func scan(row pgx.Row) (*domain.CustomField, error) {
	var f domain.CustomField
	err := row.Scan(&f.ID, &f.WorkspaceID, &f.ListID, &f.Name, &f.Kind, &f.Config, &f.Required, &f.OrderIndex, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repo) Create(ctx context.Context, f *domain.CustomField) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO custom_fields (workspace_id, list_id, name, kind, config, required, order_index)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at
	`, f.WorkspaceID, f.ListID, f.Name, f.Kind, f.Config, f.Required, f.OrderIndex).
		Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.CustomField, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM custom_fields WHERE id=$1`, id))
}

func (r *Repo) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.CustomField, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM custom_fields WHERE list_id=$1 ORDER BY order_index, created_at`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return readMany(rows)
}

func (r *Repo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.CustomField, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM custom_fields WHERE workspace_id=$1 AND list_id IS NULL ORDER BY order_index, created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return readMany(rows)
}

func readMany(rows pgx.Rows) ([]domain.CustomField, error) {
	var out []domain.CustomField
	for rows.Next() {
		var f domain.CustomField
		if err := rows.Scan(&f.ID, &f.WorkspaceID, &f.ListID, &f.Name, &f.Kind, &f.Config, &f.Required, &f.OrderIndex, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, f *domain.CustomField) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE custom_fields
		SET name=$2, kind=$3, config=$4, required=$5, order_index=$6, updated_at=NOW()
		WHERE id=$1
	`, f.ID, f.Name, f.Kind, f.Config, f.Required, f.OrderIndex)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM custom_fields WHERE id=$1`, id)
	return err
}
