package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WorkspaceUser is a User enriched with the workspace-level role/membership
// metadata the User Management page needs. Returned by UserRepo.ListByWorkspace
// so the UI can render role chips alongside name/email without a second
// round-trip.
type WorkspaceUser struct {
	User
	Role         string    `json:"role"`
	JoinedAt     time.Time `json:"joined_at"`
	IsOwner      bool      `json:"is_owner"` // workspaces.owner_id == User.ID
}

type UserRepo interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)

	// User Management surface. Operations are scoped to the workspace the
	// caller has user.manage on; the policy layer enforces that — the
	// repo just executes the queries.
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceUser, error)
	UpdateName(ctx context.Context, id uuid.UUID, name string) error
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Global admin surface — backs /api/v1/admin/users. Listed users may
	// belong to zero workspaces (newly-created accounts that haven't been
	// invited anywhere yet). limit is clamped 1..200 by the repo; offset
	// is forwarded as-is.
	List(ctx context.Context, limit, offset int) ([]User, error)
	Count(ctx context.Context) (int, error)
}
