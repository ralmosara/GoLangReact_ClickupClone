package credential

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

func (r *Repo) Create(ctx context.Context, c *domain.Credential) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO credentials (workspace_id, user_id, name, url, username, password_ciphertext, password_nonce, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, c.WorkspaceID, c.UserID, c.Name, c.URL, c.Username, c.PasswordCiphertext, c.PasswordNonce, c.Notes,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// GetByID is scoped to (workspace_id, user_id): returning nil for a row owned
// by another user or in a different workspace is indistinguishable from
// "doesn't exist", so an attacker can't probe for existence via this endpoint.
func (r *Repo) GetByID(ctx context.Context, workspaceID, userID, id uuid.UUID) (*domain.Credential, error) {
	var c domain.Credential
	err := r.pool.QueryRow(ctx, `
		SELECT id, workspace_id, user_id, name, url, username, password_ciphertext, password_nonce, notes, created_at, updated_at
		FROM credentials WHERE id = $1 AND user_id = $2 AND workspace_id = $3
	`, id, userID, workspaceID).Scan(
		&c.ID, &c.WorkspaceID, &c.UserID, &c.Name, &c.URL, &c.Username,
		&c.PasswordCiphertext, &c.PasswordNonce, &c.Notes, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repo) ListByUserInWorkspace(ctx context.Context, workspaceID, userID uuid.UUID) ([]domain.Credential, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, workspace_id, user_id, name, url, username, password_ciphertext, password_nonce, notes, created_at, updated_at
		FROM credentials WHERE user_id = $1 AND workspace_id = $2
		ORDER BY name
	`, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Credential
	for rows.Next() {
		var c domain.Credential
		if err := rows.Scan(
			&c.ID, &c.WorkspaceID, &c.UserID, &c.Name, &c.URL, &c.Username,
			&c.PasswordCiphertext, &c.PasswordNonce, &c.Notes, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, c *domain.Credential) error {
	return r.pool.QueryRow(ctx, `
		UPDATE credentials
		SET name=$4, url=$5, username=$6, password_ciphertext=$7, password_nonce=$8, notes=$9, updated_at=NOW()
		WHERE id=$1 AND user_id=$2 AND workspace_id=$3
		RETURNING updated_at
	`, c.ID, c.UserID, c.WorkspaceID, c.Name, c.URL, c.Username, c.PasswordCiphertext, c.PasswordNonce, c.Notes,
	).Scan(&c.UpdatedAt)
}

func (r *Repo) Delete(ctx context.Context, workspaceID, userID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM credentials WHERE id=$1 AND user_id=$2 AND workspace_id=$3`, id, userID, workspaceID)
	return err
}
