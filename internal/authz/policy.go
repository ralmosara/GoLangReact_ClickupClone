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
// hierarchy by mapping them via RoleRank in the future; today we treat them as
// minimum-priv "member".
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

// Policy enforces membership and role-based access for M2+ features.
// M2 wires it into view, status, tag, member, and audit handlers.
type Policy interface {
	RequireWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) error
	RequireWorkspaceRole(ctx context.Context, workspaceID, userID uuid.UUID, minRole string) error

	RequireSpaceMember(ctx context.Context, spaceID, userID uuid.UUID) error
	RequireSpaceRole(ctx context.Context, spaceID, userID uuid.UUID, minRole string) error

	// Convenience helpers for nested entities — they resolve the owning
	// workspace/space and dispatch to the matching check.
	RequireListAccess(ctx context.Context, listID, userID uuid.UUID) error
	RequireTaskAccess(ctx context.Context, taskID, userID uuid.UUID) error
}

type policy struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) Policy { return &policy{pool: pool} }

// --- workspace --------------------------------------------------------------

func (p *policy) workspaceRole(ctx context.Context, wsID, userID uuid.UUID) (string, error) {
	var role *string
	err := p.pool.QueryRow(ctx,
		`SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`,
		wsID, userID).Scan(&role)
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

func (p *policy) RequireWorkspaceRole(ctx context.Context, wsID, userID uuid.UUID, minRole string) error {
	role, err := p.workspaceRole(ctx, wsID, userID)
	if err != nil {
		return err
	}
	if role == "" || RoleRank(role) < RoleRank(minRole) {
		return ErrForbidden
	}
	return nil
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
