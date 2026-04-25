package attachment

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

const cols = `id, task_id, uploader_id, filename, path, size_bytes, mime_type, created_at`

func (r *Repo) Create(ctx context.Context, a *domain.Attachment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO attachments (task_id, uploader_id, filename, path, size_bytes, mime_type)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at
	`, a.TaskID, a.UploaderID, a.Filename, a.Path, a.SizeBytes, a.MimeType).Scan(&a.ID, &a.CreatedAt)
}

func (r *Repo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM attachments WHERE task_id=$1 ORDER BY created_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Attachment
	for rows.Next() {
		var a domain.Attachment
		if err := rows.Scan(&a.ID, &a.TaskID, &a.UploaderID, &a.Filename, &a.Path, &a.SizeBytes, &a.MimeType, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	var a domain.Attachment
	err := r.pool.QueryRow(ctx, `SELECT `+cols+` FROM attachments WHERE id=$1`, id).
		Scan(&a.ID, &a.TaskID, &a.UploaderID, &a.Filename, &a.Path, &a.SizeBytes, &a.MimeType, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM attachments WHERE id=$1`, id)
	return err
}
