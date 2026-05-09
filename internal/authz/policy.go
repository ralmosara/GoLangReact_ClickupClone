package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrForbidden is returned when an actor does not satisfy a policy.
var ErrForbidden = errors.New("forbidden")

// Role names in descending authority. Custom roles can sit anywhere in the
// hierarchy via their `rank` column; the legacy text-name path here is
// preserved for the compatibility WHERE-clause join.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleGuest  = "guest"
)

// RoleRank returns a monotonically increasing number where higher = more power.
// Unknown roles map to member (2).
func RoleRank(r string) int {
	switch r {
	case RoleOwner:
		return 4
	case RoleAdmin:
		return 3
	case RoleMember:
		return 2
	case RoleGuest:
		return 1
	default:
		return 2
	}
}

// Policy enforces membership and permission-based access for M2+ features.
//
// As of phase 2 there are two complementary check modes:
//
//   - Rank-based  — RequireWorkspaceRole / RequireSpaceRole. Compatible with
//     pre-phase-2 callers; all built-in roles have a `rank` so this still
//     works against rows that have been migrated to role_id.
//   - Permission-based — RequirePermission / HasPermission. Goes through the
//     roles + role_permissions tables. Custom roles MUST be checked this way
//     because their rank is essentially arbitrary; checking against rank
//     would let a custom "Project Manager" role ranked above member silently
//     bypass admin-only operations.
//
// Code introduced in phase 2 should prefer permission-based checks. Existing
// rank-based checks are left in place to keep the diff bounded; they are
// equivalent for the four built-in roles.
type Policy interface {
	RequireWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) error
	RequireWorkspaceRole(ctx context.Context, workspaceID, userID uuid.UUID, minRole string) error

	RequireSpaceMember(ctx context.Context, spaceID, userID uuid.UUID) error
	RequireSpaceRole(ctx context.Context, spaceID, userID uuid.UUID, minRole string) error

	// Convenience helpers for nested entities — they resolve the owning
	// workspace/space and dispatch to the matching check.
	RequireListAccess(ctx context.Context, listID, userID uuid.UUID) error
	RequireTaskAccess(ctx context.Context, taskID, userID uuid.UUID) error

	// Permission-based — phase 2.
	RequirePermission(ctx context.Context, workspaceID, userID uuid.UUID, key string) error
	HasPermission(ctx context.Context, workspaceID, userID uuid.UUID, key string) (bool, error)

	// Cross-workspace — true when the user holds the permission on AT
	// LEAST ONE workspace. Used by global admin endpoints (e.g. the
	// /api/v1/admin/users surface) where there is no workspace in the
	// URL but we still want the same role-driven gate.
	RequirePermissionAnywhere(ctx context.Context, userID uuid.UUID, key string) error
	HasPermissionAnywhere(ctx context.Context, userID uuid.UUID, key string) (bool, error)

	// HasRoleAnywhere is a strict identity check: true when the user is
	// assigned the named role in at least one workspace. Distinct from
	// the permission check above — owners have a superset of admin
	// permissions but their *role name* is "owner", so a strict
	// HasRoleAnywhere(uid, "admin") returns false for them. Used by
	// /admin/users when the policy intent is "literally the admin role,
	// not just anyone with user.manage".
	HasRoleAnywhere(ctx context.Context, userID uuid.UUID, roleName string) (bool, error)
	RequireRoleAnywhere(ctx context.Context, userID uuid.UUID, roleName string) error
}

type policy struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) Policy { return &policy{pool: pool} }

// --- workspace --------------------------------------------------------------

// workspaceRole resolves a member's effective role string. Order:
//
//  1. workspace_members.role_id IS NOT NULL → resolve via roles.id.
//  2. otherwise fall back to the legacy text column workspace_members.role.
//
// Returns an empty string when no membership exists.
func (p *policy) workspaceRole(ctx context.Context, wsID, userID uuid.UUID) (string, error) {
	var role *string
	err := p.pool.QueryRow(ctx, `
		SELECT COALESCE(
		    (SELECT name FROM roles WHERE id = wm.role_id),
		    wm.role
		)
		  FROM workspace_members wm
		 WHERE wm.workspace_id = $1 AND wm.user_id = $2
	`, wsID, userID).Scan(&role)
	if err != nil || role == nil {
		return "", err
	}
	return *role, nil
}

func (p *policy) RequireWorkspaceMember(ctx context.Context, wsID, userID uuid.UUID) error {
	role, err := p.workspaceRole(ctx, wsID, userID)
	if err != nil {
		return err
	}
	if role == "" {
		return ErrForbidden
	}
	return nil
}

// RequireWorkspaceRole compares against the rank ladder. For the four
// built-in role names this is identical to the legacy implementation; for
// custom roles the rank set by the workspace admin determines the answer.
func (p *policy) RequireWorkspaceRole(ctx context.Context, wsID, userID uuid.UUID, minRole string) error {
	rank, err := p.workspaceRoleRank(ctx, wsID, userID)
	if err != nil {
		return err
	}
	if rank < 0 || rank < RoleRank(minRole) {
		return ErrForbidden
	}
	return nil
}

// workspaceRoleRank reads the numeric rank from the linked role row, or
// derives it from the legacy text column when role_id is null. Returns -1
// when the user is not a member.
func (p *policy) workspaceRoleRank(ctx context.Context, wsID, userID uuid.UUID) (int, error) {
	var rank *int
	err := p.pool.QueryRow(ctx, `
		SELECT COALESCE(
		    (SELECT rank FROM roles WHERE id = wm.role_id),
		    CASE wm.role
		        WHEN 'owner'  THEN 4
		        WHEN 'admin'  THEN 3
		        WHEN 'member' THEN 2
		        WHEN 'guest'  THEN 1
		        ELSE 2
		    END
		)
		  FROM workspace_members wm
		 WHERE wm.workspace_id = $1 AND wm.user_id = $2
	`, wsID, userID).Scan(&rank)
	if err != nil || rank == nil {
		return -1, err
	}
	return *rank, nil
}

// --- space ------------------------------------------------------------------

func (p *policy) spaceRole(ctx context.Context, spaceID, userID uuid.UUID) (string, error) {
	var role *string
	err := p.pool.QueryRow(ctx,
		`SELECT role FROM space_members WHERE space_id=$1 AND user_id=$2`,
		spaceID, userID).Scan(&role)
	if err != nil || role == nil {
		return "", err
	}
	return *role, nil
}

func (p *policy) spaceWorkspace(ctx context.Context, spaceID uuid.UUID) (uuid.UUID, error) {
	var wsID uuid.UUID
	err := p.pool.QueryRow(ctx, `SELECT workspace_id FROM spaces WHERE id=$1`, spaceID).Scan(&wsID)
	return wsID, err
}

func (p *policy) RequireSpaceMember(ctx context.Context, spaceID, userID uuid.UUID) error {
	// Explicit space membership satisfies this check.
	role, err := p.spaceRole(ctx, spaceID, userID)
	if err == nil && role != "" {
		return nil
	}
	// Otherwise workspace membership (default open spaces) suffices.
	wsID, err := p.spaceWorkspace(ctx, spaceID)
	if err != nil {
		return err
	}
	return p.RequireWorkspaceMember(ctx, wsID, userID)
}

func (p *policy) RequireSpaceRole(ctx context.Context, spaceID, userID uuid.UUID, minRole string) error {
	if role, err := p.spaceRole(ctx, spaceID, userID); err == nil && role != "" {
		if RoleRank(role) >= RoleRank(minRole) {
			return nil
		}
		// fall through — workspace role may still be elevated
	}
	wsID, err := p.spaceWorkspace(ctx, spaceID)
	if err != nil {
		return err
	}
	return p.RequireWorkspaceRole(ctx, wsID, userID, minRole)
}

// --- convenience nested lookups --------------------------------------------

func (p *policy) RequireListAccess(ctx context.Context, listID, userID uuid.UUID) error {
	var spaceID uuid.UUID
	err := p.pool.QueryRow(ctx, `SELECT space_id FROM lists WHERE id=$1`, listID).Scan(&spaceID)
	if err != nil {
		return err
	}
	return p.RequireSpaceMember(ctx, spaceID, userID)
}

func (p *policy) RequireTaskAccess(ctx context.Context, taskID, userID uuid.UUID) error {
	var listID uuid.UUID
	err := p.pool.QueryRow(ctx, `SELECT list_id FROM tasks WHERE id=$1`, taskID).Scan(&listID)
	if err != nil {
		return err
	}
	return p.RequireListAccess(ctx, listID, userID)
}

// --- permission-based -------------------------------------------------------

// HasPermission runs the same EXISTS query as RoleRepo.HasPermission. The
// duplication is intentional — keeping the SQL inline here avoids forcing
// every Policy consumer to also wire a *role.Repo at construction.
func (p *policy) HasPermission(ctx context.Context, wsID, userID uuid.UUID, key string) (bool, error) {
	var ok bool
	err := p.pool.QueryRow(ctx, `
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
	`, wsID, userID, key).Scan(&ok)
	return ok, err
}

func (p *policy) RequirePermission(ctx context.Context, wsID, userID uuid.UUID, key string) error {
	ok, err := p.HasPermission(ctx, wsID, userID, key)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// HasPermissionAnywhere is the same query as HasPermission but without
// the workspace filter — useful for endpoints that aren't scoped to a
// single workspace (e.g. POST /api/v1/admin/users, which creates a user
// account without attaching them to any workspace). A workspace owner
// who holds user.manage on workspace A can therefore create global user
// accounts; the new user joins no workspace until invited.
func (p *policy) HasPermissionAnywhere(ctx context.Context, userID uuid.UUID, key string) (bool, error) {
	var ok bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1
		      FROM workspace_members wm
		      JOIN roles r
		        ON r.id = COALESCE(
		             wm.role_id,
		             (SELECT id FROM roles WHERE workspace_id IS NULL AND name = wm.role)
		           )
		      JOIN role_permissions rp ON rp.role_id = r.id
		     WHERE wm.user_id        = $1
		       AND rp.permission_key = $2
		)
	`, userID, key).Scan(&ok)
	return ok, err
}

func (p *policy) RequirePermissionAnywhere(ctx context.Context, userID uuid.UUID, key string) error {
	ok, err := p.HasPermissionAnywhere(ctx, userID, key)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// HasRoleAnywhere checks for an exact role-name match across all the
// user's workspace memberships. Resolution mirrors the
// EffectivePermissions join: prefer role_id, fall back to the legacy
// text role column, and the resolved role's name must equal roleName.
//
// Owners are NOT admins by this check — their role name is "owner".
// That's the entire point: callers wanting "owner OR admin" should
// stick with RequirePermissionAnywhere(PermUserManage); callers
// wanting strict admin-only use this.
func (p *policy) HasRoleAnywhere(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	var ok bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1
		      FROM workspace_members wm
		      JOIN roles r
		        ON r.id = COALESCE(
		             wm.role_id,
		             (SELECT id FROM roles WHERE workspace_id IS NULL AND name = wm.role)
		           )
		     WHERE wm.user_id = $1
		       AND r.name     = $2
		)
	`, userID, roleName).Scan(&ok)
	return ok, err
}

func (p *policy) RequireRoleAnywhere(ctx context.Context, userID uuid.UUID, roleName string) error {
	ok, err := p.HasRoleAnywhere(ctx, userID, roleName)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}
