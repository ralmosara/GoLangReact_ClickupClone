package role

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

const roleCols = `id, workspace_id, name, description, is_builtin, rank, created_at, updated_at`

// ── permissions ──────────────────────────────────────────────────────────

func (r *Repo) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT key, category, description, created_at
		FROM permissions
		ORDER BY category, key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.Key, &p.Category, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ── roles ────────────────────────────────────────────────────────────────

func (r *Repo) ListBuiltinRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+roleCols+` FROM roles WHERE workspace_id IS NULL ORDER BY rank DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRoles(rows)
}

func (r *Repo) ListWorkspaceRoles(ctx context.Context, workspaceID uuid.UUID) ([]domain.Role, error) {
	// Returns built-in roles + the workspace's custom roles, in one list.
	rows, err := r.pool.Query(ctx, `
		SELECT `+roleCols+` FROM roles
		 WHERE workspace_id IS NULL OR workspace_id = $1
		 ORDER BY workspace_id NULLS FIRST, rank DESC, name
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRoles(rows)
}

func (r *Repo) GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+roleCols+` FROM roles WHERE id = $1`, id)
	role, err := scanRole(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return role, err
}

func (r *Repo) GetBuiltinRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+roleCols+` FROM roles WHERE workspace_id IS NULL AND name = $1`, name)
	role, err := scanRole(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return role, err
}

func (r *Repo) CreateRole(ctx context.Context, role *domain.Role) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO roles (workspace_id, name, description, is_builtin, rank)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at, updated_at
	`, role.WorkspaceID, role.Name, role.Description, role.IsBuiltin, role.Rank).
		Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
}

func (r *Repo) UpdateRole(ctx context.Context, role *domain.Role) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE roles
		   SET name        = $2,
		       description = $3,
		       rank        = $4,
		       updated_at  = NOW()
		 WHERE id = $1
		   AND is_builtin = FALSE
	`, role.ID, role.Name, role.Description, role.Rank)
	return err
}

func (r *Repo) DeleteRole(ctx context.Context, id uuid.UUID) error {
	// Built-in roles are protected by the WHERE clause; deleting one is a
	// no-op rather than an error so the caller can issue a blanket "delete
	// every custom role for workspace X" without filtering first.
	_, err := r.pool.Exec(ctx,
		`DELETE FROM roles WHERE id = $1 AND is_builtin = FALSE`, id)
	return err
}

// ── role permissions ────────────────────────────────────────────────────

func (r *Repo) ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT permission_key FROM role_permissions WHERE role_id = $1 ORDER BY permission_key`,
		roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// SetRolePermissions replaces the role's grant set atomically. Unknown
// permission keys are silently dropped — the FK to permissions(key)
// enforces validity at the DB layer, but we filter here too so a typo in a
// caller-supplied list doesn't fail the whole transaction.
func (r *Repo) SetRolePermissions(ctx context.Context, roleID uuid.UUID, keys []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return err
	}
	if len(keys) == 0 {
		return tx.Commit(ctx)
	}
	// Bulk insert with a SELECT against the catalog so unknown keys are dropped.
	if _, err := tx.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_key)
		SELECT $1, p.key
		  FROM permissions p
		 WHERE p.key = ANY($2::TEXT[])
		ON CONFLICT DO NOTHING
	`, roleID, keys); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ── authorization queries ───────────────────────────────────────────────

// EffectivePermissions returns the deduped permission set for (workspace,
// user). Resolution order:
//
//  1. workspace_members.role_id IS NOT NULL  → grants come from that role.
//  2. otherwise, fall back to the legacy text column workspace_members.role
//     and resolve via roles.name = wm.role + workspace_id IS NULL.
//
// Both paths union into one query so a single round-trip suffices.
func (r *Repo) EffectivePermissions(ctx context.Context, workspaceID, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT rp.permission_key
		  FROM workspace_members wm
		  JOIN roles r
		    ON r.id = COALESCE(
		         wm.role_id,
		         (SELECT id FROM roles WHERE workspace_id IS NULL AND name = wm.role)
		       )
		  JOIN role_permissions rp ON rp.role_id = r.id
		 WHERE wm.workspace_id = $1
		   AND wm.user_id      = $2
		 ORDER BY rp.permission_key
	`, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// RoleNamesForUser returns the role name(s) the user holds in the
// workspace. Typically one row per user but the schema allows more
// (legacy text role + assigned role_id). Mirrors the COALESCE join
// EffectivePermissions uses so the source-of-truth is identical.
//
// Surfaced by the /workspaces/{id}/roles/effective endpoint so the FE
// can gate on a strict role match (e.g. "only `admin`, not `owner`")
// without fanning out to a separate endpoint.
func (r *Repo) RoleNamesForUser(ctx context.Context, workspaceID, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT r.name
		  FROM workspace_members wm
		  JOIN roles r
		    ON r.id = COALESCE(
		         wm.role_id,
		         (SELECT id FROM roles WHERE workspace_id IS NULL AND name = wm.role)
		       )
		 WHERE wm.workspace_id = $1
		   AND wm.user_id      = $2
		 ORDER BY r.name
	`, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// HasPermission is the hot-path predicate the policy layer calls per
// request. It runs the same join as EffectivePermissions but with an
// EXISTS short-circuit so the planner skips work after the first match.
func (r *Repo) HasPermission(ctx context.Context, workspaceID, userID uuid.UUID, key string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1
		      FROM workspace_members wm
		      JOIN roles r
		        ON r.id = COALESCE(
		             wm.role_id,
		             (SELECT id FROM roles WHERE workspace_id IS NULL AND name = wm.role)
		           )
		      JOIN role_permissions rp ON rp.role_id = r.id
		     WHERE wm.workspace_id   = $1
		       AND wm.user_id        = $2
		       AND rp.permission_key = $3
		)
	`, workspaceID, userID, key).Scan(&ok)
	return ok, err
}

// AssignMemberRole sets workspace_members.role_id and keeps the legacy
// text column in sync (so old code that still selects `role` directly
// keeps reading the human-readable name).
func (r *Repo) AssignMemberRole(ctx context.Context, workspaceID, userID, roleID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE workspace_members
		   SET role_id = $3,
		       role    = COALESCE((SELECT name FROM roles WHERE id = $3), role)
		 WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID, roleID)
	return err
}

// ── helpers ──────────────────────────────────────────────────────────────

func scanRole(row pgx.Row) (*domain.Role, error) {
	var r domain.Role
	if err := row.Scan(
		&r.ID, &r.WorkspaceID, &r.Name, &r.Description,
		&r.IsBuiltin, &r.Rank, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

func scanRoles(rows pgx.Rows) ([]domain.Role, error) {
	var out []domain.Role
	for rows.Next() {
		var r domain.Role
		if err := rows.Scan(
			&r.ID, &r.WorkspaceID, &r.Name, &r.Description,
			&r.IsBuiltin, &r.Rank, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
