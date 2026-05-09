package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Permission is the catalog row. Keys are stable strings; descriptions
// can be edited freely without breaking authorization checks.
type Permission struct {
	Key         string    `json:"key"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Role is either a built-in role (WorkspaceID == nil, IsBuiltin == true)
// or a workspace-defined custom role. The Rank field is kept in sync with
// the legacy authz ladder (owner=4, admin=3, member=2, guest=1) so calls
// like RequireWorkspaceRole(minRole=member) keep working alongside the new
// permission-based checks.
type Role struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID *uuid.UUID `json:"workspace_id,omitempty"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsBuiltin   bool       `json:"is_builtin"`
	Rank        int        `json:"rank"`
	Permissions []string   `json:"permissions,omitempty"` // populated by service.GetWithPermissions
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// RoleRepo persists roles, the static permission catalog, and the
// role↔permission grants. EffectivePermissions resolves a (workspace,
// user) pair to the union of permission keys granted via the user's role.
type RoleRepo interface {
	// Permissions catalog (read-only at runtime; seeded by the migration).
	ListPermissions(ctx context.Context) ([]Permission, error)

	// Roles.
	ListBuiltinRoles(ctx context.Context) ([]Role, error)
	ListWorkspaceRoles(ctx context.Context, workspaceID uuid.UUID) ([]Role, error)
	GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error)
	GetBuiltinRoleByName(ctx context.Context, name string) (*Role, error)
	CreateRole(ctx context.Context, r *Role) error
	UpdateRole(ctx context.Context, r *Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error

	// Role ↔ permission grants.
	ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error)
	SetRolePermissions(ctx context.Context, roleID uuid.UUID, keys []string) error

	// Authorization queries.
	EffectivePermissions(ctx context.Context, workspaceID, userID uuid.UUID) ([]string, error)
	HasPermission(ctx context.Context, workspaceID, userID uuid.UUID, key string) (bool, error)
	// RoleNamesForUser returns the user's assigned role name(s) in the
	// workspace. Used by the FE to gate on a strict role match (e.g.
	// "only `admin`, not `owner`") without leaking permission detail.
	RoleNamesForUser(ctx context.Context, workspaceID, userID uuid.UUID) ([]string, error)

	// Membership wiring — used when assigning a user a role.
	AssignMemberRole(ctx context.Context, workspaceID, userID, roleID uuid.UUID) error
}
