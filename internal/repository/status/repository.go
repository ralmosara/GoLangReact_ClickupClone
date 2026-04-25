package status

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

const cols = `id, list_id, name, color, category, order_index, created_at`

func (r *Repo) Create(ctx context.Context, s *domain.Status) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO statuses (list_id, name, color, category, order_index)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at
	`, s.ListID, s.Name, s.Color, s.Category, s.OrderIndex).Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Status, error) {
	var s domain.Status
	err := r.pool.QueryRow(ctx, `SELECT `+cols+` FROM statuses WHERE id=$1`, id).
		Scan(&s.ID, &s.ListID, &s.Name, &s.Color, &s.Category, &s.OrderIndex, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.Status, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM statuses WHERE list_id=$1 ORDER BY order_index, created_at`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Status
	for rows.Next() {
		var s domain.Status
		if err := rows.Scan(&s.ID, &s.ListID, &s.Name, &s.Color, &s.Category, &s.OrderIndex, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, s *domain.Status) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE statuses SET name=$2, color=$3, category=$4, order_index=$5 WHERE id=$1
	`, s.ID, s.Name, s.Color, s.Category, s.OrderIndex)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM statuses WHERE id=$1`, id)
	return err
}
