package session

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const cols = `id, user_id, label, user_agent, ip, created_at, last_seen_at, expires_at`

func (r *Repo) Create(ctx context.Context, s *domain.Session) error {
	var ip any
	if s.IP != nil {
		ip = s.IP.String()
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO user_sessions (id, user_id, label, user_agent, ip, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, last_seen_at
	`, s.ID, s.UserID, s.Label, s.UserAgent, ip, s.ExpiresAt).
		Scan(&s.CreatedAt, &s.LastSeenAt)
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+cols+` FROM user_sessions WHERE id = $1`, id)
	s, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *Repo) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Session, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+cols+` FROM user_sessions
		WHERE user_id = $1 AND expires_at > now()
		ORDER BY last_seen_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Session
	for rows.Next() {
		s, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repo) Touch(ctx context.Context, id uuid.UUID, olderThan time.Duration) error {
	// Conditional UPDATE keeps the table's hot-write rate bounded — at
	// olderThan=1m we cap at 1 write/device/minute regardless of req rate.
	//
	// Compute the cutoff in Go rather than relying on Postgres to parse
	// time.Duration's "1m0s" format as INTERVAL — it doesn't, and the
	// resulting parse error would surface as a Touch failure. Cleaner to
	// just pass a timestamp.
	cutoff := time.Now().Add(-olderThan)
	_, err := r.pool.Exec(ctx, `
		UPDATE user_sessions
		   SET last_seen_at = now()
		 WHERE id = $1
		   AND last_seen_at < $2
	`, id, cutoff)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM user_sessions WHERE id = $1`, id)
	return err
}

func (r *Repo) DeleteAllExcept(ctx context.Context, userID, keepID uuid.UUID) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM user_sessions WHERE user_id = $1 AND id <> $2`,
		userID, keepID,
	)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *Repo) DeleteExpired(ctx context.Context) (int, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM user_sessions WHERE expires_at <= now()`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// rowScanner is the common shape between pgx.Row and pgx.Rows so the same
// scan helper covers both Get and ListByUser.
type rowScanner interface {
	Scan(dest ...any) error
}

func scan(r rowScanner) (*domain.Session, error) {
	var (
		s     domain.Session
		ipStr *string
	)
	if err := r.Scan(&s.ID, &s.UserID, &s.Label, &s.UserAgent, &ipStr, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt); err != nil {
		return nil, err
	}
	if ipStr != nil && *ipStr != "" {
		// pgx returns inet as text in default; parse to netip for the
		// domain. Bad rows degrade to nil rather than crashing.
		if a, err := netip.ParseAddr(*ipStr); err == nil {
			s.IP = &a
		}
	}
	return &s, nil
}
