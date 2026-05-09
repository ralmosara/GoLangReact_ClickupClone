package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OAuthIdentity links one of our users to an external IdP subject. The same
// user can have multiple identities (Google + Microsoft). The (provider,
// subject) pair is unique — one external account maps to exactly one user.
type OAuthIdentity struct {
	ID         uuid.UUID       `json:"id"`
	UserID     uuid.UUID       `json:"user_id"`
	Provider   string          `json:"provider"`
	Subject    string          `json:"subject"`
	Email      *string         `json:"email,omitempty"`
	RawProfile json.RawMessage `json:"-"` // not exposed; useful for debugging
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type OAuthIdentityRepo interface {
	GetByProviderSubject(ctx context.Context, provider, subject string) (*OAuthIdentity, error)
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*OAuthIdentity, error)
	Upsert(ctx context.Context, ident *OAuthIdentity) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]OAuthIdentity, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
