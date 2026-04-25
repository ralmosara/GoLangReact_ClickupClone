package goaltarget

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

const cols = `id, goal_id, name, kind, target_number, current_number, currency, target_boolean,
	current_boolean, task_list_id, order_index, created_at, updated_at`

func scan(row pgx.Row) (*domain.GoalTarget, error) {
	var t domain.GoalTarget
	err := row.Scan(&t.ID, &t.GoalID, &t.Name, &t.Kind, &t.TargetNumber, &t.CurrentNumber, &t.Currency,
		&t.TargetBoolean, &t.CurrentBoolean, &t.TaskListID, &t.OrderIndex, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) Create(ctx context.Context, t *domain.GoalTarget) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO goal_targets (goal_id, name, kind, target_number, current_number, currency,
			target_boolean, current_boolean, task_list_id, order_index)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at, updated_at
	`, t.GoalID, t.Name, t.Kind, t.TargetNumber, t.CurrentNumber, t.Currency,
		t.TargetBoolean, t.CurrentBoolean, t.TaskListID, t.OrderIndex).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.GoalTarget, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM goal_targets WHERE id=$1`, id))
}

func (r *Repo) ListByGoal(ctx context.Context, goalID uuid.UUID) ([]domain.GoalTarget, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM goal_targets WHERE goal_id=$1 ORDER BY order_index, created_at`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.GoalTarget
	for rows.Next() {
		var t domain.GoalTarget
		if err := rows.Scan(&t.ID, &t.GoalID, &t.Name, &t.Kind, &t.TargetNumber, &t.CurrentNumber,
			&t.Currency, &t.TargetBoolean, &t.CurrentBoolean, &t.TaskListID, &t.OrderIndex,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, t *domain.GoalTarget) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE goal_targets SET
			name=$2, kind=$3, target_number=$4, current_number=$5, currency=$6,
			target_boolean=$7, current_boolean=$8, task_list_id=$9, order_index=$10,
			updated_at=NOW()
		WHERE id=$1
	`, t.ID, t.Name, t.Kind, t.TargetNumber, t.CurrentNumber, t.Currency,
		t.TargetBoolean, t.CurrentBoolean, t.TaskListID, t.OrderIndex)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM goal_targets WHERE id=$1`, id)
	return err
}

// TaskCompletion returns (completed, total) for non-archived, non-subtask
// tasks in a list. A task counts as completed when either the legacy
// `status = 'completed'` is true OR a done-category status maps to it.
func (r *Repo) TaskCompletion(ctx context.Context, listID uuid.UUID) (int, int, error) {
	var total, completed int
	err := r.pool.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE t.archived = FALSE AND t.parent_task_id IS NULL) AS total,
		  COUNT(*) FILTER (
		    WHERE t.archived = FALSE AND t.parent_task_id IS NULL AND (
		      t.status = 'completed'
		      OR t.completed_at IS NOT NULL
		      OR (t.status_id IS NOT NULL AND EXISTS (
		            SELECT 1 FROM statuses s
		            WHERE s.id = t.status_id AND s.category IN ('done','closed')))
		    )
		  ) AS completed
		FROM tasks t WHERE t.list_id = $1
	`, listID).Scan(&total, &completed)
	return completed, total, err
}
