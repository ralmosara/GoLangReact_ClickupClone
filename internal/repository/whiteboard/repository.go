package whiteboard

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

const cols = `id, workspace_id, space_id, name, snapshot, version, creator_id, created_at, updated_at`

func scan(row pgx.Row) (*domain.Whiteboard, error) {
	var w domain.Whiteboard
	err := row.Scan(&w.ID, &w.WorkspaceID, &w.SpaceID, &w.Name, &w.Snapshot, &w.Version,
		&w.CreatorID, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func scanMany(rows pgx.Rows) ([]domain.Whiteboard, error) {
	var out []domain.Whiteboard
	for rows.Next() {
		var w domain.Whiteboard
		if err := rows.Scan(&w.ID, &w.WorkspaceID, &w.SpaceID, &w.Name, &w.Snapshot, &w.Version,
			&w.CreatorID, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, w *domain.Whiteboard) error {
	snap := w.Snapshot
	if len(snap) == 0 {
		snap = []byte("{}")
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO whiteboards (workspace_id, space_id, name, snapshot, creator_id)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, version, created_at, updated_at
	`, w.WorkspaceID, w.SpaceID, w.Name, snap, w.CreatorID).
		Scan(&w.ID, &w.Version, &w.CreatedAt, &w.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Whiteboard, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM whiteboards WHERE id=$1`, id))
}

func (r *Repo) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Whiteboard, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM whiteboards WHERE workspace_id=$1 ORDER BY created_at DESC`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]domain.Whiteboard, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM whiteboards WHERE space_id=$1 ORDER BY created_at DESC`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) Update(ctx context.Context, w *domain.Whiteboard) error {
	snap := w.Snapshot
	if len(snap) == 0 {
		snap = []byte("{}")
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE whiteboards SET name=$2, snapshot=$3, version=version+1, updated_at=NOW() WHERE id=$1`,
		w.ID, w.Name, snap)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM whiteboards WHERE id=$1`, id)
	return err
}
