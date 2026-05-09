package user

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

func (r *Repo) Create(ctx context.Context, u *domain.User) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, u.Email, u.PasswordHash, u.Name).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, name, avatar_url, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, name, avatar_url, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ListByWorkspace returns every user that's a member of the given
// workspace, joined with their membership role. Sorted by created_at so
// re-renders are stable. The COALESCE on roles.name reads the role string
// from the FK row when available (post-migration-031) and falls back to
// the legacy text column for any rows that pre-date the back-fill.
func (r *Repo) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.WorkspaceUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.email, u.password_hash, u.name, u.avatar_url, u.created_at, u.updated_at,
		       COALESCE((SELECT name FROM roles WHERE id = wm.role_id), wm.role) AS role,
		       wm.created_at AS joined_at,
		       (w.owner_id = u.id) AS is_owner
		  FROM workspace_members wm
		  JOIN users      u ON u.id = wm.user_id
		  JOIN workspaces w ON w.id = wm.workspace_id
		 WHERE wm.workspace_id = $1
		 ORDER BY wm.created_at
	`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.WorkspaceUser
	for rows.Next() {
		var wu domain.WorkspaceUser
		if err := rows.Scan(
			&wu.ID, &wu.Email, &wu.PasswordHash, &wu.Name, &wu.AvatarURL, &wu.CreatedAt, &wu.UpdatedAt,
			&wu.Role, &wu.JoinedAt, &wu.IsOwner,
		); err != nil {
			return nil, err
		}
		out = append(out, wu)
	}
	return out, rows.Err()
}

func (r *Repo) UpdateName(ctx context.Context, id uuid.UUID, name string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET name = $2, updated_at = NOW() WHERE id = $1`, id, name)
	return err
}

func (r *Repo) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, id, hash)
	return err
}

// Delete hard-deletes a user. FKs handle the cascade for us:
//
//   - workspace_members, space_members, mfa_secrets, mfa_recovery_codes,
//     oauth_identities, channel_members, task_assignees → ON DELETE CASCADE
//   - workspaces.owner_id, comments.author_id, attachments.uploader_id,
//     audit_log.actor_id, tasks.assignee_id/creator_id, messages.author_id,
//     statuses (no FK), notifications.user_id (CASCADE) → ON DELETE SET NULL
//     where the row should survive the user
//
// A real deployment that needs an audit trail across deletions should
// flip this to soft-delete; see "Out of scope" in the iteration plan.
func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// List returns a single page of every user in the system. Newest first
// so the global User Management page shows recent creations at the top.
// Pagination is plain limit/offset — fine at the scale this app targets;
// switch to keyset pagination if the table grows past low-millions.
func (r *Repo) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, email, password_hash, name, avatar_url, created_at, updated_at
		FROM users
		ORDER BY created_at DESC, id
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}
