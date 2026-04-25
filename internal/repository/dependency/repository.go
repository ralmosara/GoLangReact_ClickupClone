package dependency

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Create(ctx context.Context, d *domain.TaskDependency) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO task_dependencies (task_id, depends_on_id, kind)
		VALUES ($1,$2,$3)
		ON CONFLICT (task_id, depends_on_id) DO UPDATE SET kind = EXCLUDED.kind
		RETURNING created_at
	`, d.TaskID, d.DependsOnID, d.Kind).Scan(&d.CreatedAt)
}

func (r *Repo) Delete(ctx context.Context, taskID, dependsOnID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM task_dependencies WHERE task_id=$1 AND depends_on_id=$2`,
		taskID, dependsOnID)
	return err
}

func (r *Repo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT task_id, depends_on_id, kind, created_at FROM task_dependencies WHERE task_id=$1 ORDER BY created_at`,
		taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return readMany(rows)
}

func (r *Repo) ListDependents(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT task_id, depends_on_id, kind, created_at FROM task_dependencies WHERE depends_on_id=$1 ORDER BY created_at`,
		taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return readMany(rows)
}

// WouldCreateCycle walks the reachable set starting from `dependsOnID`
// along "waits on" edges. If we reach `taskID`, adding `taskID waits on
// dependsOnID` would close a cycle. The recursive CTE depth-limits at 100
// to bound pathological cases.
func (r *Repo) WouldCreateCycle(ctx context.Context, taskID, dependsOnID uuid.UUID) (bool, error) {
	if taskID == dependsOnID {
		return true, nil
	}
	var reachable bool
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE reach AS (
			SELECT depends_on_id AS node, 1 AS depth
			FROM task_dependencies
			WHERE task_id = $1
			UNION
			SELECT d.depends_on_id, r.depth + 1
			FROM task_dependencies d
			JOIN reach r ON d.task_id = r.node
			WHERE r.depth < 100
		)
		SELECT EXISTS (SELECT 1 FROM reach WHERE node = $2)
	`, dependsOnID, taskID).Scan(&reachable)
	return reachable, err
}

func readMany(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]domain.TaskDependency, error) {
	var out []domain.TaskDependency
	for rows.Next() {
		var d domain.TaskDependency
		if err := rows.Scan(&d.TaskID, &d.DependsOnID, &d.Kind, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
