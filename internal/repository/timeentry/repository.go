package timeentry

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

const cols = `id, task_id, user_id, started_at, stopped_at, duration_s, note, billable, created_at, updated_at`

func scan(row pgx.Row) (*domain.TimeEntry, error) {
	var e domain.TimeEntry
	err := row.Scan(&e.ID, &e.TaskID, &e.UserID, &e.StartedAt, &e.StoppedAt, &e.DurationS, &e.Note, &e.Billable, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repo) Create(ctx context.Context, e *domain.TimeEntry) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO time_entries (task_id, user_id, started_at, stopped_at, duration_s, note, billable)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at
	`, e.TaskID, e.UserID, e.StartedAt, e.StoppedAt, e.DurationS, e.Note, e.Billable).
		Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.TimeEntry, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM time_entries WHERE id=$1`, id))
}

func (r *Repo) Update(ctx context.Context, e *domain.TimeEntry) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE time_entries
		SET stopped_at=$2, duration_s=$3, note=$4, billable=$5, updated_at=NOW()
		WHERE id=$1
	`, e.ID, e.StoppedAt, e.DurationS, e.Note, e.Billable)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM time_entries WHERE id=$1`, id)
	return err
}

func (r *Repo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.TimeEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM time_entries WHERE task_id=$1 ORDER BY started_at DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return readMany(rows)
}

func (r *Repo) ActiveForUser(ctx context.Context, userID uuid.UUID) ([]domain.TimeEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM time_entries WHERE user_id=$1 AND stopped_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return readMany(rows)
}

func readMany(rows pgx.Rows) ([]domain.TimeEntry, error) {
	var out []domain.TimeEntry
	for rows.Next() {
		var e domain.TimeEntry
		if err := rows.Scan(&e.ID, &e.TaskID, &e.UserID, &e.StartedAt, &e.StoppedAt, &e.DurationS, &e.Note, &e.Billable, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repo) Report(ctx context.Context, f domain.TimeEntryFilter) ([]domain.TimeReportBucket, error) {
	var where []string
	var args []any
	i := 1
	add := func(clause string, v any) {
		where = append(where, strings.ReplaceAll(clause, "?", "$"+strconv.Itoa(i)))
		args = append(args, v)
		i++
	}
	// Default: only include stopped entries so in-progress timers don't double-count.
	where = append(where, "te.stopped_at IS NOT NULL")
	if f.WorkspaceID != nil {
		add(`t.list_id IN (
			SELECT id FROM lists WHERE space_id IN (
				SELECT id FROM spaces WHERE workspace_id = ?
			)
		)`, *f.WorkspaceID)
	}
	if f.TaskID != nil {
		add("te.task_id = ?", *f.TaskID)
	}
	if f.UserID != nil {
		add("te.user_id = ?", *f.UserID)
	}
	if f.From != nil {
		add("te.started_at >= ?", *f.From)
	}
	if f.To != nil {
		add("te.started_at < ?", *f.To)
	}
	clause := ""
	if len(where) > 0 {
		clause = "WHERE " + strings.Join(where, " AND ")
	}
	q := `
		SELECT te.user_id,
		       COALESCE(u.email, '') AS email,
		       COALESCE(u.name, '')  AS name,
		       COUNT(DISTINCT te.task_id) AS task_count,
		       COALESCE(SUM(te.duration_s), 0)::bigint AS total_s,
		       COALESCE(SUM(CASE WHEN te.billable THEN te.duration_s ELSE 0 END), 0)::bigint AS billable_s
		FROM time_entries te
		JOIN tasks t ON t.id = te.task_id
		LEFT JOIN users u ON u.id = te.user_id
		` + clause + `
		GROUP BY te.user_id, u.email, u.name
		ORDER BY total_s DESC
	`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TimeReportBucket
	for rows.Next() {
		var b domain.TimeReportBucket
		if err := rows.Scan(&b.UserID, &b.Email, &b.Name, &b.TaskCount, &b.TotalS, &b.BillableS); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
