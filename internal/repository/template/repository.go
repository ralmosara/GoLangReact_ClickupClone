package template

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

const cols = `id, workspace_id, kind, name, description, snapshot, creator_id, created_at, updated_at`

func scan(row pgx.Row) (*domain.Template, error) {
	var t domain.Template
	err := row.Scan(&t.ID, &t.WorkspaceID, &t.Kind, &t.Name, &t.Description, &t.Snapshot,
		&t.CreatorID, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) Create(ctx context.Context, t *domain.Template) error {
	snap := t.Snapshot
	if len(snap) == 0 {
		snap = []byte("{}")
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO templates (workspace_id, kind, name, description, snapshot, creator_id)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at, updated_at
	`, t.WorkspaceID, t.Kind, t.Name, t.Description, snap, t.CreatorID).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM templates WHERE id=$1`, id))
}

func (r *Repo) ListByWorkspace(ctx context.Context, wsID uuid.UUID, kind string) ([]domain.Template, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if kind != "" {
		rows, err = r.pool.Query(ctx, `SELECT `+cols+` FROM templates WHERE workspace_id=$1 AND kind=$2 ORDER BY name`, wsID, kind)
	} else {
		rows, err = r.pool.Query(ctx, `SELECT `+cols+` FROM templates WHERE workspace_id=$1 ORDER BY kind, name`, wsID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Template
	for rows.Next() {
		var t domain.Template
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Kind, &t.Name, &t.Description, &t.Snapshot,
			&t.CreatorID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, t *domain.Template) error {
	snap := t.Snapshot
	if len(snap) == 0 {
		snap = []byte("{}")
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE templates SET name=$2, description=$3, snapshot=$4, updated_at=NOW() WHERE id=$1`,
		t.ID, t.Name, t.Description, snap)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM templates WHERE id=$1`, id)
	return err
}
