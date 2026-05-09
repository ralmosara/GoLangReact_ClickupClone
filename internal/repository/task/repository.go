package task

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const taskCols = `id, list_id, parent_task_id, name, description, status, status_id, priority, position,
	assignee_id, creator_id, due_at, start_at, completed_at, archived,
	recurring_rule, recurring_parent_id, sprint_id, points, time_estimate_s, created_at, updated_at`

func scanTask(row pgx.Row) (*domain.Task, error) {
	var t domain.Task
	err := row.Scan(&t.ID, &t.ListID, &t.ParentTaskID, &t.Name, &t.Description, &t.Status, &t.StatusID, &t.Priority, &t.Position,
		&t.AssigneeID, &t.CreatorID, &t.DueAt, &t.StartAt, &t.CompletedAt, &t.Archived,
		&t.RecurringRule, &t.RecurringParentID, &t.SprintID, &t.Points, &t.EstimateSeconds, &t.CreatedAt, &t.UpdatedAt)
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
			&t.RecurringRule, &t.RecurringParentID, &t.SprintID, &t.Points, &t.EstimateSeconds, &t.CreatedAt, &t.UpdatedAt); err != nil {
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
			assignee_id, creator_id, due_at, start_at, recurring_rule, recurring_parent_id, sprint_id, points, time_estimate_s)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING id, created_at, updated_at
	`,
		t.ListID, t.ParentTaskID, t.Name, t.Description, t.Status, t.StatusID, t.Priority, t.Position,
		t.AssigneeID, t.CreatorID, t.DueAt, t.StartAt, t.RecurringRule, t.RecurringParentID, t.SprintID, t.Points, t.EstimateSeconds,
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
			sprint_id=$17, points=$18, time_estimate_s=$19,
			updated_at=NOW()
		WHERE id=$1
	`,
		t.ID, t.Name, t.Description, t.Status, t.StatusID, t.Priority, t.Position,
		t.AssigneeID, t.DueAt, t.StartAt, t.CompletedAt, t.Archived,
		t.ListID, t.ParentTaskID,
		t.RecurringRule, t.RecurringParentID,
		t.SprintID, t.Points, t.EstimateSeconds,
	)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id=$1`, id)
	return err
}

func (r *Repo) Reorder(ctx context.Context, id uuid.UUID, status string, statusID *uuid.UUID, position float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET status=$2, status_id=$3, position=$4, updated_at=NOW() WHERE id=$1`,
		id, status, statusID, position)
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

// WorkloadBucket is one assignee's slice of the workload report. Open
// is "tasks not yet completed within the window"; CompletedSeconds is
// the sum of estimates for completed tasks (defaults to time-tracked
// duration when no estimate exists; both are zero when neither does).
type WorkloadBucket struct {
	UserID            uuid.UUID `json:"user_id"`
	OpenTaskCount     int       `json:"open_task_count"`
	OpenEstimateSec   int       `json:"open_estimate_seconds"`
	CompletedTasks    int       `json:"completed_task_count"`
	CompletedEstSec   int       `json:"completed_estimate_seconds"`
	OverdueTasks      int       `json:"overdue_task_count"`
}

// Workload aggregates current assignment load per user. The window is
// (from, to) and applies to due_at (open) / completed_at (closed).
//
// Tasks with no assignee land in the special-case zero-uuid bucket so
// the UI can render an "Unassigned" row alongside named users.
//
// Estimates default to zero per task — the FE typically converts to
// hours for display. The intentional shape: "for the period [from..to],
// who is on the hook for what?"
func (r *Repo) Workload(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]WorkloadBucket, error) {
	const q = `
		WITH scoped AS (
		  SELECT t.*
		    FROM tasks t
		    JOIN lists l  ON l.id = t.list_id
		    JOIN spaces s ON s.id = l.space_id
		   WHERE s.workspace_id = $1
		     AND t.archived = FALSE
		     AND t.parent_task_id IS NULL
		     AND (t.due_at IS NULL OR t.due_at < $3)
		     AND (t.completed_at IS NULL OR t.completed_at >= $2)
		)
		SELECT
		  COALESCE(assignee_id, '00000000-0000-0000-0000-000000000000'::uuid)         AS user_id,
		  COUNT(*) FILTER (WHERE completed_at IS NULL)::int                            AS open_count,
		  COALESCE(SUM(time_estimate_s) FILTER (WHERE completed_at IS NULL), 0)::int   AS open_est,
		  COUNT(*) FILTER (WHERE completed_at IS NOT NULL)::int                        AS done_count,
		  COALESCE(SUM(time_estimate_s) FILTER (WHERE completed_at IS NOT NULL), 0)::int AS done_est,
		  COUNT(*) FILTER (WHERE completed_at IS NULL AND due_at IS NOT NULL AND due_at < NOW())::int AS overdue
		  FROM scoped
		 GROUP BY 1
		 ORDER BY open_est DESC, open_count DESC`
	rows, err := r.pool.Query(ctx, q, workspaceID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorkloadBucket
	for rows.Next() {
		var b WorkloadBucket
		if err := rows.Scan(&b.UserID, &b.OpenTaskCount, &b.OpenEstimateSec,
			&b.CompletedTasks, &b.CompletedEstSec, &b.OverdueTasks); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ListByAssigneeInWorkspace returns up to N open tasks (unarchived,
// not completed) assigned to assigneeID anywhere in the workspace,
// ordered by due_at ASC NULLS LAST. Used by the My Tasks dashboard
// widget; the FE buckets by overdue / today / this_week / later.
func (r *Repo) ListByAssigneeInWorkspace(ctx context.Context, workspaceID, assigneeID uuid.UUID, limit int) ([]domain.Task, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+taskCols+`
		  FROM tasks t
		  JOIN lists l  ON l.id = t.list_id
		  JOIN spaces s ON s.id = l.space_id
		 WHERE s.workspace_id = $1
		   AND t.assignee_id  = $2
		   AND t.archived     = FALSE
		   AND t.completed_at IS NULL
		   AND t.parent_task_id IS NULL
		 ORDER BY t.due_at ASC NULLS LAST, t.created_at DESC
		 LIMIT $3
	`, workspaceID, assigneeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRows(rows)
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
