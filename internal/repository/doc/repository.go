package doc

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

const cols = `id, workspace_id, space_id, parent_id, creator_id, title, icon, content, content_text,
	version, archived, order_index, created_at, updated_at`

func scan(row pgx.Row) (*domain.Doc, error) {
	var d domain.Doc
	err := row.Scan(&d.ID, &d.WorkspaceID, &d.SpaceID, &d.ParentID, &d.CreatorID, &d.Title, &d.Icon,
		&d.Content, &d.ContentText, &d.Version, &d.Archived, &d.OrderIndex, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func scanMany(rows pgx.Rows) ([]domain.Doc, error) {
	var out []domain.Doc
	for rows.Next() {
		var d domain.Doc
		if err := rows.Scan(&d.ID, &d.WorkspaceID, &d.SpaceID, &d.ParentID, &d.CreatorID, &d.Title, &d.Icon,
			&d.Content, &d.ContentText, &d.Version, &d.Archived, &d.OrderIndex, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, d *domain.Doc) error {
	content := d.Content
	if len(content) == 0 {
		content = []byte("{}")
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO docs (workspace_id, space_id, parent_id, creator_id, title, icon, content, content_text, order_index)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, version, created_at, updated_at
	`, d.WorkspaceID, d.SpaceID, d.ParentID, d.CreatorID, d.Title, d.Icon, content, d.ContentText, d.OrderIndex).
		Scan(&d.ID, &d.Version, &d.CreatedAt, &d.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Doc, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM docs WHERE id=$1`, id))
}

func (r *Repo) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Doc, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM docs WHERE workspace_id=$1 AND archived = FALSE ORDER BY order_index, created_at`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) ListChildren(ctx context.Context, parentID uuid.UUID) ([]domain.Doc, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM docs WHERE parent_id=$1 AND archived = FALSE ORDER BY order_index, created_at`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) Update(ctx context.Context, d *domain.Doc) error {
	content := d.Content
	if len(content) == 0 {
		content = []byte("{}")
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE docs SET
			title=$2, icon=$3, content=$4, content_text=$5, parent_id=$6, space_id=$7,
			archived=$8, order_index=$9, version = version + 1, updated_at=NOW()
		WHERE id=$1
	`, d.ID, d.Title, d.Icon, content, d.ContentText, d.ParentID, d.SpaceID, d.Archived, d.OrderIndex)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM docs WHERE id=$1`, id)
	return err
}
