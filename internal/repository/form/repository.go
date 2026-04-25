package form

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

const cols = `id, list_id, name, description, fields, is_public, submit_count, creator_id, created_at, updated_at`

func scan(row pgx.Row) (*domain.Form, error) {
	var f domain.Form
	err := row.Scan(&f.ID, &f.ListID, &f.Name, &f.Description, &f.Fields, &f.IsPublic, &f.SubmitCount,
		&f.CreatorID, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repo) Create(ctx context.Context, f *domain.Form) error {
	fields := f.Fields
	if len(fields) == 0 {
		fields = []byte("[]")
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO forms (list_id, name, description, fields, is_public, creator_id)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, submit_count, created_at, updated_at
	`, f.ListID, f.Name, f.Description, fields, f.IsPublic, f.CreatorID).
		Scan(&f.ID, &f.SubmitCount, &f.CreatedAt, &f.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Form, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM forms WHERE id=$1`, id))
}

func (r *Repo) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.Form, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM forms WHERE list_id=$1 ORDER BY created_at DESC`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Form
	for rows.Next() {
		var f domain.Form
		if err := rows.Scan(&f.ID, &f.ListID, &f.Name, &f.Description, &f.Fields, &f.IsPublic, &f.SubmitCount,
			&f.CreatorID, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, f *domain.Form) error {
	fields := f.Fields
	if len(fields) == 0 {
		fields = []byte("[]")
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE forms SET name=$2, description=$3, fields=$4, is_public=$5, updated_at=NOW()
		WHERE id=$1
	`, f.ID, f.Name, f.Description, fields, f.IsPublic)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM forms WHERE id=$1`, id)
	return err
}

func (r *Repo) RecordSubmission(ctx context.Context, sub *domain.FormSubmission) error {
	payload := sub.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	// Use a single statement to insert + bump counter so the counter stays consistent
	// even if Create tasks fails and we don't link task_id.
	err := r.pool.QueryRow(ctx, `
		WITH inserted AS (
		  INSERT INTO form_submissions (form_id, task_id, payload, submitter_ip)
		  VALUES ($1, $2, $3, $4)
		  RETURNING id, submitted_at
		),
		bump AS (
		  UPDATE forms SET submit_count = submit_count + 1, updated_at = NOW() WHERE id = $1
		)
		SELECT id, submitted_at FROM inserted
	`, sub.FormID, sub.TaskID, payload, sub.SubmitterIP).Scan(&sub.ID, &sub.SubmittedAt)
	return err
}

func (r *Repo) ListSubmissions(ctx context.Context, formID uuid.UUID, limit int) ([]domain.FormSubmission, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, form_id, task_id, payload, submitted_at, COALESCE(submitter_ip, '')
		FROM form_submissions WHERE form_id=$1 ORDER BY submitted_at DESC LIMIT $2
	`, formID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FormSubmission
	for rows.Next() {
		var s domain.FormSubmission
		if err := rows.Scan(&s.ID, &s.FormID, &s.TaskID, &s.Payload, &s.SubmittedAt, &s.SubmitterIP); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
