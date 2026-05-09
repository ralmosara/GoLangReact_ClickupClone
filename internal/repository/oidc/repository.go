package oidc

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

const cols = `id, user_id, provider, subject, email, raw_profile, created_at, updated_at`

func (r *Repo) GetByProviderSubject(ctx context.Context, provider, subject string) (*domain.OAuthIdentity, error) {
	var ident domain.OAuthIdentity
	err := r.pool.QueryRow(ctx, `
		SELECT `+cols+` FROM oauth_identities
		WHERE provider = $1 AND subject = $2
	`, provider, subject).Scan(
		&ident.ID, &ident.UserID, &ident.Provider, &ident.Subject,
		&ident.Email, &ident.RawProfile, &ident.CreatedAt, &ident.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ident, nil
}

func (r *Repo) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*domain.OAuthIdentity, error) {
	var ident domain.OAuthIdentity
	err := r.pool.QueryRow(ctx, `
		SELECT `+cols+` FROM oauth_identities
		WHERE user_id = $1 AND provider = $2
	`, userID, provider).Scan(
		&ident.ID, &ident.UserID, &ident.Provider, &ident.Subject,
		&ident.Email, &ident.RawProfile, &ident.CreatedAt, &ident.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ident, nil
}

func (r *Repo) Upsert(ctx context.Context, ident *domain.OAuthIdentity) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO oauth_identities (user_id, provider, subject, email, raw_profile)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (provider, subject) DO UPDATE
		   SET email       = EXCLUDED.email,
		       raw_profile = EXCLUDED.raw_profile,
		       updated_at  = NOW()
		RETURNING id, created_at, updated_at
	`, ident.UserID, ident.Provider, ident.Subject, ident.Email, ident.RawProfile).Scan(
		&ident.ID, &ident.CreatedAt, &ident.UpdatedAt,
	)
}

func (r *Repo) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.OAuthIdentity, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+cols+` FROM oauth_identities
		WHERE user_id = $1
		ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OAuthIdentity
	for rows.Next() {
		var ident domain.OAuthIdentity
		if err := rows.Scan(
			&ident.ID, &ident.UserID, &ident.Provider, &ident.Subject,
			&ident.Email, &ident.RawProfile, &ident.CreatedAt, &ident.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, ident)
	}
	return out, rows.Err()
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM oauth_identities WHERE id = $1`, id)
	return err
}
