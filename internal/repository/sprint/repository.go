package sprint

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const cols = `id, list_id, name, starts_at, ends_at, goal_points, status, created_at, updated_at`

func scan(row pgx.Row) (*domain.Sprint, error) {
	var s domain.Sprint
	err := row.Scan(&s.ID, &s.ListID, &s.Name, &s.StartsAt, &s.EndsAt, &s.GoalPoints, &s.Status,
		&s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) Create(ctx context.Context, s *domain.Sprint) error {
	status := s.Status
	if status == "" {
		status = domain.SprintStatusPlanned
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO sprints (list_id, name, starts_at, ends_at, goal_points, status)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at, updated_at
	`, s.ListID, s.Name, s.StartsAt, s.EndsAt, s.GoalPoints, status).
		Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM sprints WHERE id=$1`, id))
}

func (r *Repo) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.Sprint, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM sprints WHERE list_id=$1 ORDER BY starts_at`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Sprint
	for rows.Next() {
		var s domain.Sprint
		if err := rows.Scan(&s.ID, &s.ListID, &s.Name, &s.StartsAt, &s.EndsAt, &s.GoalPoints,
			&s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, s *domain.Sprint) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sprints SET name=$2, starts_at=$3, ends_at=$4, goal_points=$5, status=$6, updated_at=NOW()
		WHERE id=$1
	`, s.ID, s.Name, s.StartsAt, s.EndsAt, s.GoalPoints, s.Status)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sprints WHERE id=$1`, id)
	return err
}

// MoveOpenTasks reassigns all non-completed tasks from one sprint to another.
// A zero destination UUID detaches them (sprint_id = NULL) so the backlog isn't
// orphaned when there's no follow-on sprint.
func (r *Repo) MoveOpenTasks(ctx context.Context, fromSprintID, toSprintID uuid.UUID) (int, error) {
	var dest any = toSprintID
	if toSprintID == uuid.Nil {
		dest = nil
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET sprint_id = $2, updated_at = NOW()
		WHERE sprint_id = $1
		  AND completed_at IS NULL
		  AND status <> 'completed'
	`, fromSprintID, dest)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// NextOpenSprint returns the earliest sprint with status != 'closed' whose
// `starts_at` is on or after `after`.
func (r *Repo) NextOpenSprint(ctx context.Context, listID uuid.UUID, after time.Time) (*domain.Sprint, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+cols+` FROM sprints
		WHERE list_id = $1 AND status <> 'closed' AND starts_at >= $2
		ORDER BY starts_at
		LIMIT 1
	`, listID, after)
	return scan(row)
}

// VelocityRow is one row of velocity data — completed + goal points per sprint.
// Kept private to this file; main.go adapts it to the dashboard service shape.
type VelocityRow struct {
	SprintID   uuid.UUID
	Name       string
	EndsAt     time.Time
	Completed  int
	GoalPoints int
}

func (r *Repo) Velocity(ctx context.Context, listID uuid.UUID, limit int) ([]VelocityRow, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.pool.Query(ctx, `
		SELECT sp.id, sp.name, sp.ends_at, sp.goal_points,
		       COALESCE(SUM(t.points) FILTER (WHERE t.completed_at IS NOT NULL), 0)::int AS completed
		FROM sprints sp
		LEFT JOIN tasks t ON t.sprint_id = sp.id
		WHERE sp.list_id = $1
		GROUP BY sp.id
		ORDER BY sp.ends_at DESC
		LIMIT $2
	`, listID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VelocityRow
	for rows.Next() {
		var v VelocityRow
		if err := rows.Scan(&v.SprintID, &v.Name, &v.EndsAt, &v.GoalPoints, &v.Completed); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// BurndownSeries is a daily total→completed→remaining summary for the sprint
// period. Uses generate_series so days with no completions still emit a row.
func (r *Repo) BurndownSeries(ctx context.Context, sprintID uuid.UUID) ([]domain.BurndownPoint, error) {
	rows, err := r.pool.Query(ctx, `
		WITH sp AS (SELECT * FROM sprints WHERE id = $1),
		     days AS (
		       SELECT generate_series(
		         date_trunc('day', (SELECT starts_at FROM sp)),
		         date_trunc('day', (SELECT ends_at   FROM sp)),
		         '1 day'::interval
		       ) AS day
		     ),
		     total AS (
		       SELECT COALESCE(SUM(points), 0)::int AS tp
		       FROM tasks WHERE sprint_id = $1
		     )
		SELECT d.day,
		       (SELECT tp FROM total) AS total_points,
		       COALESCE((
		         SELECT SUM(points)::int FROM tasks
		         WHERE sprint_id = $1
		           AND completed_at IS NOT NULL
		           AND completed_at <= d.day + interval '1 day'
		       ), 0) AS completed_points
		FROM days d
		ORDER BY d.day
	`, sprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BurndownPoint
	for rows.Next() {
		var p domain.BurndownPoint
		var total, completed int
		if err := rows.Scan(&p.Day, &total, &completed); err != nil {
			return nil, err
		}
		p.TotalPoints = total
		p.CompletedPoints = completed
		p.RemainingPoints = total - completed
		if p.RemainingPoints < 0 {
			p.RemainingPoints = 0
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
