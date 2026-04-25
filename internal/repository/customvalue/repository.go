package customvalue

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Upsert(ctx context.Context, v *domain.CustomValue) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO task_custom_values (task_id, field_id, value)
		VALUES ($1,$2,$3)
		ON CONFLICT (task_id, field_id) DO UPDATE SET value = EXCLUDED.value
	`, v.TaskID, v.FieldID, v.Value)
	return err
}

func (r *Repo) Delete(ctx context.Context, taskID, fieldID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM task_custom_values WHERE task_id=$1 AND field_id=$2`, taskID, fieldID)
	return err
}

func (r *Repo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.CustomValue, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT task_id, field_id, value FROM task_custom_values WHERE task_id=$1`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CustomValue
	for rows.Next() {
		var v domain.CustomValue
		if err := rows.Scan(&v.TaskID, &v.FieldID, &v.Value); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
