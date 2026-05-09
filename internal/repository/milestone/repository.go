package milestone

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

// scanCols projects the milestone row plus the task aggregates so the
// UI gets "12 of 30 done" without an N+1.
const aggregatedSelect = `
	SELECT m.id, m.workspace_id, m.name, m.description, m.color,
	       m.due_at, m.status, m.owner_id, m.created_at, m.updated_at,
	       COALESCE(agg.task_count, 0)::int      AS task_count,
	       COALESCE(agg.completed_count, 0)::int AS completed_count
	  FROM milestones m
	  LEFT JOIN (
	    SELECT mt.milestone_id,
	           COUNT(*) AS task_count,
	           COUNT(*) FILTER (WHERE t.completed_at IS NOT NULL) AS completed_count
	      FROM milestone_tasks mt
	      JOIN tasks t ON t.id = mt.task_id
	     GROUP BY mt.milestone_id
	  ) agg ON agg.milestone_id = m.id`

func (r *Repo) Create(ctx context.Context, m *domain.Milestone) error {
	if m.Status == "" {
		m.Status = domain.MilestonePlanned
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO milestones (workspace_id, name, description, color, due_at, status, owner_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, m.WorkspaceID, m.Name, m.Description, m.Color, m.DueAt, string(m.Status), m.OwnerID).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (*domain.Milestone, error) {
	row := r.pool.QueryRow(ctx, aggregatedSelect+` WHERE m.id = $1`, id)
	m, err := scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *Repo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Milestone, error) {
	rows, err := r.pool.Query(ctx, aggregatedSelect+`
		WHERE m.workspace_id = $1
		ORDER BY m.due_at ASC, m.created_at ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Milestone
	for rows.Next() {
		m, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, m *domain.Milestone) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE milestones
		   SET name        = $2,
		       description = $3,
		       color       = $4,
		       due_at      = $5,
		       status      = $6,
		       owner_id    = $7,
		       updated_at  = NOW()
		 WHERE id = $1
	`, m.ID, m.Name, m.Description, m.Color, m.DueAt, string(m.Status), m.OwnerID)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM milestones WHERE id = $1`, id)
	return err
}

func (r *Repo) AttachTask(ctx context.Context, milestoneID, taskID, actor uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO milestone_tasks (milestone_id, task_id, added_by)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, milestoneID, taskID, actor)
	return err
}

func (r *Repo) DetachTask(ctx context.Context, milestoneID, taskID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM milestone_tasks WHERE milestone_id = $1 AND task_id = $2`,
		milestoneID, taskID,
	)
	return err
}

func (r *Repo) ListTasks(ctx context.Context, milestoneID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT task_id FROM milestone_tasks
		 WHERE milestone_id = $1
		 ORDER BY added_at ASC
	`, milestoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanRow(r rowScanner) (*domain.Milestone, error) {
	var (
		m            domain.Milestone
		statusStr    string
	)
	if err := r.Scan(
		&m.ID, &m.WorkspaceID, &m.Name, &m.Description, &m.Color,
		&m.DueAt, &statusStr, &m.OwnerID, &m.CreatedAt, &m.UpdatedAt,
		&m.TaskCount, &m.CompletedTasks,
	); err != nil {
		return nil, err
	}
	m.Status = domain.MilestoneStatus(statusStr)
	return &m, nil
}
