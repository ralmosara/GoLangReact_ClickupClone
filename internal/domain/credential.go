package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Credential struct {
	ID                 uuid.UUID `json:"id"`
	WorkspaceID        uuid.UUID `json:"workspace_id"`
	UserID             uuid.UUID `json:"user_id"`
	Name               string    `json:"name"`
	URL                string    `json:"url"`
	Username           string    `json:"username"`
	PasswordCiphertext []byte    `json:"-"`
	PasswordNonce      []byte    `json:"-"`
	// Password is populated by the service only on detail fetch (after decrypt).
	// It is omitted from list responses and from create/update responses.
	Password  string    `json:"password,omitempty"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CredentialRepo interface {
	Create(ctx context.Context, c *Credential) error
	GetByID(ctx context.Context, workspaceID, userID, id uuid.UUID) (*Credential, error)
	ListByUserInWorkspace(ctx context.Context, workspaceID, userID uuid.UUID) ([]Credential, error)
	Update(ctx context.Context, c *Credential) error
	Delete(ctx context.Context, workspaceID, userID, id uuid.UUID) error
}
