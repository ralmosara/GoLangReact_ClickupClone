package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Member is a user's role attachment to a workspace or a space.
type Member struct {
	WorkspaceID *uuid.UUID `json:"workspace_id,omitempty"`
	SpaceID     *uuid.UUID `json:"space_id,omitempty"`
	UserID      uuid.UUID  `json:"user_id"`
	Email       string     `json:"email,omitempty"`
	Name        string     `json:"name,omitempty"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	Role        string     `json:"role"`
	CreatedAt   time.Time  `json:"created_at"`
}

type MemberRepo interface {
	// Workspace membership
	AddWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, role string) error
	RemoveWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) error
	ListWorkspaceMembers(ctx context.Context, workspaceID uuid.UUID) ([]Member, error)
	IsWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)

	// Space membership
	AddSpaceMember(ctx context.Context, spaceID, userID uuid.UUID, role string) error
	RemoveSpaceMember(ctx context.Context, spaceID, userID uuid.UUID) error
	ListSpaceMembers(ctx context.Context, spaceID uuid.UUID) ([]Member, error)
	IsSpaceMember(ctx context.Context, spaceID, userID uuid.UUID) (bool, error)
}
