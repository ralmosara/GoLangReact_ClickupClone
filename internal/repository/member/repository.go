package member

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// --- workspace membership -----------------------------------------------------

func (r *Repo) AddWorkspaceMember(ctx context.Context, wsID, userID uuid.UUID, role string) error {
	if role == "" {
		role = "member"
	}
	// Populate role_id alongside the legacy text column. The sub-SELECT
	// resolves the built-in role by name; if `role` is a custom-role name
	// scoped to the workspace, the LIMIT 1 picks up that match. If neither
	// exists, role_id stays NULL and the COALESCE in policy.workspaceRole
	// keeps reads working off the text column.
	_, err := r.pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, role_id)
		VALUES (
		    $1, $2, $3,
		    (SELECT id FROM roles
		      WHERE name = $3
		        AND (workspace_id IS NULL OR workspace_id = $1)
		      ORDER BY workspace_id NULLS LAST
		      LIMIT 1)
		)
		ON CONFLICT (workspace_id, user_id) DO UPDATE
		   SET role    = EXCLUDED.role,
		       role_id = EXCLUDED.role_id
	`, wsID, userID, role)
	return err
}

func (r *Repo) RemoveWorkspaceMember(ctx context.Context, wsID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wsID, userID)
	return err
}

func (r *Repo) ListWorkspaceMembers(ctx context.Context, wsID uuid.UUID) ([]domain.Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT wm.workspace_id, wm.user_id, u.email, u.name, u.avatar_url, wm.role, wm.created_at
		FROM workspace_members wm JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1
		ORDER BY wm.created_at
	`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Member
	for rows.Next() {
		var m domain.Member
		var wsID uuid.UUID
		if err := rows.Scan(&wsID, &m.UserID, &m.Email, &m.Name, &m.AvatarURL, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.WorkspaceID = &wsID
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repo) IsWorkspaceMember(ctx context.Context, wsID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2)`,
		wsID, userID).Scan(&ok)
	return ok, err
}

// --- space membership ---------------------------------------------------------

func (r *Repo) AddSpaceMember(ctx context.Context, spaceID, userID uuid.UUID, role string) error {
	if role == "" {
		role = "member"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO space_members (space_id, user_id, role)
		VALUES ($1,$2,$3)
		ON CONFLICT (space_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, spaceID, userID, role)
	return err
}

func (r *Repo) RemoveSpaceMember(ctx context.Context, spaceID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM space_members WHERE space_id=$1 AND user_id=$2`, spaceID, userID)
	return err
}

func (r *Repo) ListSpaceMembers(ctx context.Context, spaceID uuid.UUID) ([]domain.Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sm.space_id, sm.user_id, u.email, u.name, u.avatar_url, sm.role, sm.created_at
		FROM space_members sm JOIN users u ON u.id = sm.user_id
		WHERE sm.space_id = $1
		ORDER BY sm.created_at
	`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Member
	for rows.Next() {
		var m domain.Member
		var sID uuid.UUID
		if err := rows.Scan(&sID, &m.UserID, &m.Email, &m.Name, &m.AvatarURL, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.SpaceID = &sID
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repo) IsSpaceMember(ctx context.Context, spaceID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM space_members WHERE space_id=$1 AND user_id=$2)`,
		spaceID, userID).Scan(&ok)
	return ok, err
}
