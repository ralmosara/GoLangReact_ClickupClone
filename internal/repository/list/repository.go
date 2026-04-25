package list

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

func (r *Repo) Create(ctx context.Context, l *domain.List) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO lists (space_id, folder_id, name, position)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, l.SpaceID, l.FolderID, l.Name, l.Position).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.List, error) {
	var l domain.List
	err := r.pool.QueryRow(ctx, `
		SELECT id, space_id, folder_id, name, position, archived, created_at, updated_at
		FROM lists WHERE id = $1
	`, id).Scan(&l.ID, &l.SpaceID, &l.FolderID, &l.Name, &l.Position, &l.Archived, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repo) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]domain.List, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, space_id, folder_id, name, position, archived, created_at, updated_at
		FROM lists WHERE space_id=$1 AND archived=FALSE
		ORDER BY position, created_at
	`, spaceID)
	if err != nil {
		return nil, err
	}
	return scanLists(rows)
}

func (r *Repo) ListByFolder(ctx context.Context, folderID uuid.UUID) ([]domain.List, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, space_id, folder_id, name, position, archived, created_at, updated_at
		FROM lists WHERE folder_id=$1 AND archived=FALSE
		ORDER BY position, created_at
	`, folderID)
	if err != nil {
		return nil, err
	}
	return scanLists(rows)
}

func scanLists(rows pgx.Rows) ([]domain.List, error) {
	defer rows.Close()
	var out []domain.List
	for rows.Next() {
		var l domain.List
		if err := rows.Scan(&l.ID, &l.SpaceID, &l.FolderID, &l.Name, &l.Position, &l.Archived, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, l *domain.List) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE lists SET name=$2, folder_id=$3, position=$4, archived=$5, updated_at=NOW()
		WHERE id=$1
	`, l.ID, l.Name, l.FolderID, l.Position, l.Archived)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM lists WHERE id=$1`, id)
	return err
}
