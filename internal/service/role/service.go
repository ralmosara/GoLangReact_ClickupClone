// Package role manages built-in roles (read-only at runtime) and per-
// workspace custom roles (full CRUD), plus the role↔permission grant set.
//
// Authorization checks live in the authz package — this service only owns
// the data model. Anything that wants to know "can user X do thing Y in
// workspace Z" goes through authz.Policy.RequirePermission, which delegates
// to RoleRepo.HasPermission internally.
package role

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

var (
	ErrNotFound        = errors.New("role: not found")
	ErrBuiltinReadOnly = errors.New("role: built-in roles are read-only")
	ErrNameRequired    = errors.New("role: name required")
	ErrInvalidPerm     = errors.New("role: unknown permission key")
)

type Service struct {
	repo domain.RoleRepo
}

func New(repo domain.RoleRepo) *Service { return &Service{repo: repo} }

// ListPermissions surfaces the catalog so the UI can render a permission
// matrix when editing a custom role.
func (s *Service) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

// ListWorkspaceRoles returns built-in roles + the workspace's custom roles
// in one list. Built-ins come first.
func (s *Service) ListWorkspaceRoles(ctx context.Context, workspaceID uuid.UUID) ([]domain.Role, error) {
	roles, err := s.repo.ListWorkspaceRoles(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	// Inline each role's permissions so a single network round-trip from
	// the UI gives it everything it needs to render the matrix.
	for i := range roles {
		perms, err := s.repo.ListRolePermissions(ctx, roles[i].ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = perms
	}
	return roles, nil
}

// GetWithPermissions fetches a single role and its grant list.
func (s *Service) GetWithPermissions(ctx context.Context, roleID uuid.UUID) (*domain.Role, error) {
	r, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil || r == nil {
		if r == nil && err == nil {
			return nil, ErrNotFound
		}
		return nil, err
	}
	perms, err := s.repo.ListRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	r.Permissions = perms
	return r, nil
}

type CreateInput struct {
	WorkspaceID uuid.UUID
	Name        string
	Description string
	Rank        int      // optional — defaults to 2 (member tier)
	Permissions []string // optional — empty = no grants until first SetPermissions
}

// Create persists a new custom role for a workspace. Built-in roles must
// be seeded via the migration; this method always sets is_builtin = false
// and workspace_id = input.WorkspaceID.
func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Role, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrNameRequired
	}
	if in.Rank == 0 {
		in.Rank = 2
	}
	if err := s.validatePermissionKeys(ctx, in.Permissions); err != nil {
		return nil, err
	}
	wsID := in.WorkspaceID
	r := &domain.Role{
		WorkspaceID: &wsID,
		Name:        in.Name,
		Description: in.Description,
		IsBuiltin:   false,
		Rank:        in.Rank,
	}
	if err := s.repo.CreateRole(ctx, r); err != nil {
		return nil, err
	}
	if len(in.Permissions) > 0 {
		if err := s.repo.SetRolePermissions(ctx, r.ID, in.Permissions); err != nil {
			return nil, err
		}
		r.Permissions = in.Permissions
	}
	return r, nil
}

type UpdateInput struct {
	Name        *string
	Description *string
	Rank        *int
}

func (s *Service) Update(ctx context.Context, roleID uuid.UUID, in UpdateInput) (*domain.Role, error) {
	r, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrNotFound
	}
	if r.IsBuiltin {
		return nil, ErrBuiltinReadOnly
	}
	if in.Name != nil {
		r.Name = strings.TrimSpace(*in.Name)
		if r.Name == "" {
			return nil, ErrNameRequired
		}
	}
	if in.Description != nil {
		r.Description = *in.Description
	}
	if in.Rank != nil {
		r.Rank = *in.Rank
	}
	if err := s.repo.UpdateRole(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Service) Delete(ctx context.Context, roleID uuid.UUID) error {
	r, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrNotFound
	}
	if r.IsBuiltin {
		return ErrBuiltinReadOnly
	}
	return s.repo.DeleteRole(ctx, roleID)
}

// SetPermissions atomically replaces the role's grant set. Built-in roles
// are protected — their grants come from the migration and stay frozen.
func (s *Service) SetPermissions(ctx context.Context, roleID uuid.UUID, keys []string) error {
	r, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrNotFound
	}
	if r.IsBuiltin {
		return ErrBuiltinReadOnly
	}
	if err := s.validatePermissionKeys(ctx, keys); err != nil {
		return err
	}
	return s.repo.SetRolePermissions(ctx, roleID, keys)
}

// AssignToMember points workspace_members.role_id at the given role for
// the (workspace, user) pair. The role must belong either to the workspace
// or be a built-in.
func (s *Service) AssignToMember(ctx context.Context, workspaceID, userID, roleID uuid.UUID) error {
	r, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrNotFound
	}
	if r.WorkspaceID != nil && *r.WorkspaceID != workspaceID {
		return errors.New("role: belongs to a different workspace")
	}
	return s.repo.AssignMemberRole(ctx, workspaceID, userID, roleID)
}

func (s *Service) EffectivePermissions(ctx context.Context, workspaceID, userID uuid.UUID) ([]string, error) {
	return s.repo.EffectivePermissions(ctx, workspaceID, userID)
}

// RoleNamesForUser surfaces the user's assigned role name(s). Pass-through
// to the repo — kept on the service so handlers don't reach into the repo
// directly.
func (s *Service) RoleNamesForUser(ctx context.Context, workspaceID, userID uuid.UUID) ([]string, error) {
	return s.repo.RoleNamesForUser(ctx, workspaceID, userID)
}

// validatePermissionKeys cross-checks against the catalog so we reject
// bogus keys at the service layer with a useful error rather than relying
// on the FK to swallow them silently in SetRolePermissions.
func (s *Service) validatePermissionKeys(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	catalog, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return err
	}
	known := make(map[string]struct{}, len(catalog))
	for _, p := range catalog {
		known[p.Key] = struct{}{}
	}
	for _, k := range keys {
		if _, ok := known[k]; !ok {
			return errors.Join(ErrInvalidPerm, errors.New(k))
		}
	}
	return nil
}
