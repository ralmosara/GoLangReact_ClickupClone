package mfa

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

func (r *Repo) GetSecret(ctx context.Context, userID uuid.UUID) (*domain.MFASecret, error) {
	var s domain.MFASecret
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, secret, enrolled_at, last_used_at, created_at, updated_at
		FROM mfa_secrets WHERE user_id = $1
	`, userID).Scan(&s.UserID, &s.Secret, &s.EnrolledAt, &s.LastUsedAt, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpsertSecret writes a fresh TOTP secret for the user, resetting the
// enrolled state. The unique PK ensures one secret per user — re-enrolling
// overwrites the old one (and the caller is expected to wipe the old
// recovery codes via DeleteRecoveryCodes).
func (r *Repo) UpsertSecret(ctx context.Context, s *domain.MFASecret) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO mfa_secrets (user_id, secret, enrolled_at, last_used_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (user_id) DO UPDATE
		   SET secret       = EXCLUDED.secret,
		       enrolled_at  = EXCLUDED.enrolled_at,
		       last_used_at = EXCLUDED.last_used_at,
		       updated_at   = EXCLUDED.updated_at
	`, s.UserID, s.Secret, s.EnrolledAt, s.LastUsedAt, now)
	return err
}

func (r *Repo) MarkEnrolled(ctx context.Context, userID uuid.UUID, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE mfa_secrets SET enrolled_at = $2, updated_at = NOW() WHERE user_id = $1`,
		userID, at)
	return err
}

func (r *Repo) MarkLastUsed(ctx context.Context, userID uuid.UUID, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE mfa_secrets SET last_used_at = $2, updated_at = NOW() WHERE user_id = $1`,
		userID, at)
	return err
}

func (r *Repo) DeleteSecret(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM mfa_secrets WHERE user_id = $1`, userID)
	return err
}

func (r *Repo) InsertRecoveryCodes(ctx context.Context, userID uuid.UUID, hashes [][]byte) error {
	if len(hashes) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, h := range hashes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO mfa_recovery_codes (user_id, code_hash) VALUES ($1, $2)
			 ON CONFLICT (user_id, code_hash) DO NOTHING`,
			userID, h); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ConsumeRecoveryCode atomically marks a code as consumed. The compound
// WHERE clause means concurrent verifications race for the UPDATE and only
// one wins — `consumed` is true in the winner, false everywhere else.
func (r *Repo) ConsumeRecoveryCode(ctx context.Context, userID uuid.UUID, hash []byte) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE mfa_recovery_codes
		   SET consumed_at = NOW()
		 WHERE user_id     = $1
		   AND code_hash   = $2
		   AND consumed_at IS NULL
	`, userID, hash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repo) DeleteRecoveryCodes(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id = $1`, userID)
	return err
}

func (r *Repo) CountRemainingRecoveryCodes(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM mfa_recovery_codes WHERE user_id = $1 AND consumed_at IS NULL`,
		userID).Scan(&n)
	return n, err
}
