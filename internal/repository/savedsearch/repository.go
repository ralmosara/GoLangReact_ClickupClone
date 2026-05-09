package savedsearch

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const cols = `id, user_id, workspace_id, name, entity_type, query_text, filters, pinned, last_used_at, created_at, updated_at`

func (r *Repo) Create(ctx context.Context, s *domain.SavedSearch) error {
	if len(s.Filters) == 0 {
		s.Filters = json.RawMessage("[]")
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO saved_searches (user_id, workspace_id, name, entity_type, query_text, filters, pinned)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, s.UserID, s.WorkspaceID, s.Name, s.EntityType, s.QueryText, s.Filters, s.Pinned).
		Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.SavedSearch, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+cols+` FROM saved_searches WHERE id = $1`, id)
	s, err := scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *Repo) ListForUser(ctx context.Context, userID, workspaceID uuid.UUID) ([]domain.SavedSearch, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+cols+` FROM saved_searches
		WHERE user_id = $1 AND workspace_id = $2
		ORDER BY pinned DESC, last_used_at DESC NULLS LAST, created_at DESC
	`, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SavedSearch
	for rows.Next() {
		s, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, s *domain.SavedSearch) error {
	if len(s.Filters) == 0 {
		s.Filters = json.RawMessage("[]")
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE saved_searches
		   SET name        = $2,
		       entity_type = $3,
		       query_text  = $4,
		       filters     = $5,
		       pinned      = $6,
		       updated_at  = NOW()
		 WHERE id = $1
	`, s.ID, s.Name, s.EntityType, s.QueryText, s.Filters, s.Pinned)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM saved_searches WHERE id = $1`, id)
	return err
}

func (r *Repo) TouchUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE saved_searches SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

type rowScanner interface{ Scan(...any) error }

func scanRow(r rowScanner) (*domain.SavedSearch, error) {
	var s domain.SavedSearch
	if err := r.Scan(
		&s.ID, &s.UserID, &s.WorkspaceID, &s.Name, &s.EntityType, &s.QueryText,
		&s.Filters, &s.Pinned, &s.LastUsedAt, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &s, nil
}
