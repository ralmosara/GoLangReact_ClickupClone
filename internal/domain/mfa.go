package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MFASecret is the per-user TOTP shared secret. EnrolledAt nil means the
// user generated a secret but hasn't yet proven possession by submitting a
// valid code — login must still succeed without MFA in that intermediate
// state, and the secret should be shown to the user as a QR code so they
// can scan it again. Once EnrolledAt is set, every login requires a TOTP
// code (or a recovery code).
type MFASecret struct {
	UserID     uuid.UUID  `json:"user_id"`
	Secret     string     `json:"-"` // base32-encoded; never returned over the wire
	EnrolledAt *time.Time `json:"enrolled_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// MFARecoveryCode is a single-use code printed once during enrollment.
// CodeHash is sha256 of the plaintext; the plaintext is shown to the user
// only at generation time and is unrecoverable thereafter.
type MFARecoveryCode struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	CodeHash   []byte
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// MFARepo persists TOTP secrets and recovery codes. ConsumeRecoveryCode
// must be atomic (UPDATE ... WHERE consumed_at IS NULL) so two concurrent
// requests can't both succeed with the same code.
type MFARepo interface {
	GetSecret(ctx context.Context, userID uuid.UUID) (*MFASecret, error)
	UpsertSecret(ctx context.Context, s *MFASecret) error
	MarkEnrolled(ctx context.Context, userID uuid.UUID, at time.Time) error
	MarkLastUsed(ctx context.Context, userID uuid.UUID, at time.Time) error
	DeleteSecret(ctx context.Context, userID uuid.UUID) error

	InsertRecoveryCodes(ctx context.Context, userID uuid.UUID, hashes [][]byte) error
	ConsumeRecoveryCode(ctx context.Context, userID uuid.UUID, hash []byte) (consumed bool, err error)
	DeleteRecoveryCodes(ctx context.Context, userID uuid.UUID) error
	CountRemainingRecoveryCodes(ctx context.Context, userID uuid.UUID) (int, error)
}
