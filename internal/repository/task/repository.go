package task

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const taskCols = `id, list_id, parent_task_id, name, description, status, status_id, priority, position,
	assignee_id, creator_id, due_at, start_at, completed_at, archived,
	recurring_rule, recurring_parent_id, sprint_id, points, created_at, updated_at`

func scanTask(row pgx.Row) (*domain.Task, error) {
	var t domain.Task
	err := row.Scan(&t.ID, &t.ListID, &t.ParentTaskID, &t.Name, &t.Description, &t.Status, &t.StatusID, &t.Priority, &t.Position,
		&t.AssigneeID, &t.CreatorID, &t.DueAt, &t.StartAt, &t.CompletedAt, &t.Archived,
		&t.RecurringRule, &t.RecurringParentID, &t.SprintID, &t.Points, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func scanTaskRows(rows pgx.Rows) ([]domain.Task, error) {
	var out []domain.Task
	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.ListID, &t.ParentTaskID, &t.Name, &t.Description, &t.Status, &t.StatusID, &t.Priority, &t.Position,
			&t.AssigneeID, &t.CreatorID, &t.DueAt, &t.StartAt, &t.CompletedAt, &t.Archived,
			&t.RecurringRule, &t.RecurringParentID, &t.SprintID, &t.Points, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, t *domain.Task) error {
	if t.Position == 0 {
		max, err := r.MaxPosition(ctx, t.ListID, t.StatusID)
		if err == nil {
			t.Position = max + 1000
		} else {
			t.Position = 1000
		}
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO tasks (list_id, parent_task_id, name, description, status, status_id, priority, position,
			assignee_id, creator_id, due_at, start_at, recurring_rule, recurring_parent_id, sprint_id, points)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		RETURNING id, created_at, updated_at
	`,
		t.ListID, t.ParentTaskID, t.Name, t.Description, t.Status, t.StatusID, t.Priority, t.Position,
		t.AssigneeID, t.CreatorID, t.DueAt, t.StartAt, t.RecurringRule, t.RecurringParentID, t.SprintID, t.Points,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+taskCols+` FROM tasks WHERE id=$1`, id)
	return scanTask(row)
}

func (r *Repo) List(ctx context.Context, f domain.TaskFilter) ([]domain.Task, error) {
	var where []string
	var args []any
	i := 1
	add := func(clause string, v any) {
		where = append(where, strings.ReplaceAll(clause, "?", "$"+strconv.Itoa(i)))
		args = append(args, v)
		i++
	}
	if f.ListID != nil {
		add("list_id = ?", *f.ListID)
	}
	if f.AssigneeID != nil {
		add("assignee_id = ?", *f.AssigneeID)
	}
	if f.Status != nil {
		add("status = ?", *f.Status)
	}
	if f.StatusID != nil {
		add("status_id = ?", *f.StatusID)
	}
	if f.ParentTaskID != nil {
		add("parent_task_id = ?", *f.ParentTaskID)
	} else if !f.IncludeSubs {
		where = append(where, "parent_task_id IS NULL")
	}
	if f.Archived != nil {
		add("archived = ?", *f.Archived)
	} else {
		where = append(where, "archived = FALSE")
	}

	q := `SELECT ` + taskCols + ` FROM tasks`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY position, created_at"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func (r *Repo) ListSubtasks(ctx context.Context, parentID uuid.UUID) ([]domain.Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+taskCols+` FROM tasks WHERE parent_task_id = $1 AND archived = FALSE ORDER BY position, created_at`,
		parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func (r *Repo) Update(ctx context.Context, t *domain.Task) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tasks SET
			name=$2, description=$3, status=$4, status_id=$5, priority=$6, position=$7,
			assignee_id=$8, due_at=$9, start_at=$10, completed_at=$11, archived=$12,
			list_id=$13, parent_task_id=$14,
			recurring_rule=$15, recurring_parent_id=$16,
			sprint_id=$17, points=$18,
			updated_at=NOW()
		WHERE id=$1
	`,
		t.ID, t.Name, t.Description, t.Status, t.StatusID, t.Priority, t.Position,
		t.AssigneeID, t.DueAt, t.StartAt, t.CompletedAt, t.Archived,
		t.ListID, t.ParentTaskID,
		t.RecurringRule, t.RecurringParentID,
		t.SprintID, t.Points,
	)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id=$1`, id)
	return err
}

func (r *Repo) Reorder(ctx context.Context, id uuid.UUID, statusID *uuid.UUID, position float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET status_id=$2, position=$3, updated_at=NOW() WHERE id=$1`,
		id, statusID, position)
	return err
}

func (r *Repo) SetArchived(ctx context.Context, id uuid.UUID, archived bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET archived=$2, updated_at=NOW() WHERE id=$1`,
		id, archived)
	return err
}

func (r *Repo) MaxPosition(ctx context.Context, listID uuid.UUID, statusID *uuid.UUID) (float64, error) {
	var max *float64
	err := r.pool.QueryRow(ctx,
		`SELECT MAX(position) FROM tasks WHERE list_id=$1 AND status_id IS NOT DISTINCT FROM $2`,
		listID, statusID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if max == nil {
		return 0, nil
	}
	return *max, nil
}

// CountByStatus returns (statusName, count) rows for a list. Used by the
// task-count dashboard widget.
func (r *Repo) CountByStatus(ctx context.Context, listID uuid.UUID) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(s.name, t.status) AS name, COUNT(*)
		FROM tasks t LEFT JOIN statuses s ON s.id = t.status_id
		WHERE t.list_id = $1 AND t.archived = FALSE AND t.parent_task_id IS NULL
		GROUP BY COALESCE(s.name, t.status)
		ORDER BY 2 DESC
	`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		out[name] = count
	}
	return out, rows.Err()
}
