package view

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

const cols = `id, list_id, space_id, name, kind, config, created_at`

func scan(row pgx.Row) (*domain.View, error) {
	var v domain.View
	err := row.Scan(&v.ID, &v.ListID, &v.SpaceID, &v.Name, &v.Kind, &v.Config, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repo) Create(ctx context.Context, v *domain.View) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO views (list_id, space_id, name, kind, config)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at
	`, v.ListID, v.SpaceID, v.Name, v.Kind, v.Config).Scan(&v.ID, &v.CreatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.View, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+cols+` FROM views WHERE id=$1`, id)
	return scan(row)
}

func (r *Repo) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.View, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM views WHERE list_id=$1 ORDER BY created_at`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.View
	for rows.Next() {
		var v domain.View
		if err := rows.Scan(&v.ID, &v.ListID, &v.SpaceID, &v.Name, &v.Kind, &v.Config, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repo) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]domain.View, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM views WHERE space_id=$1 ORDER BY created_at`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.View
	for rows.Next() {
		var v domain.View
		if err := rows.Scan(&v.ID, &v.ListID, &v.SpaceID, &v.Name, &v.Kind, &v.Config, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, v *domain.View) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE views SET name=$2, kind=$3, config=$4 WHERE id=$1`,
		v.ID, v.Name, v.Kind, v.Config)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM views WHERE id=$1`, id)
	return err
}
