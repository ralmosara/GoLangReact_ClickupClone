package folder

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

func (r *Repo) Create(ctx context.Context, f *domain.Folder) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO folders (space_id, name, position)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, f.SpaceID, f.Name, f.Position).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Folder, error) {
	var f domain.Folder
	err := r.pool.QueryRow(ctx, `
		SELECT id, space_id, name, position, archived, created_at, updated_at
		FROM folders WHERE id = $1
	`, id).Scan(&f.ID, &f.SpaceID, &f.Name, &f.Position, &f.Archived, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repo) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]domain.Folder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, space_id, name, position, archived, created_at, updated_at
		FROM folders WHERE space_id=$1 AND archived=FALSE
		ORDER BY position, created_at
	`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Folder
	for rows.Next() {
		var f domain.Folder
		if err := rows.Scan(&f.ID, &f.SpaceID, &f.Name, &f.Position, &f.Archived, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, f *domain.Folder) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE folders SET name=$2, position=$3, archived=$4, updated_at=NOW()
		WHERE id=$1
	`, f.ID, f.Name, f.Position, f.Archived)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM folders WHERE id=$1`, id)
	return err
}
