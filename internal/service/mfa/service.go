// Package mfa implements TOTP (RFC 6238) multi-factor auth on top of the
// existing password login. The flow is:
//
//  1. User POSTs /me/mfa/enroll → service generates a fresh secret, returns
//     it as a base32 string + an otpauth:// URL the frontend renders as a
//     QR code. The DB row is created with enrolled_at = NULL.
//  2. User scans the QR with an authenticator app and POSTs
//     /me/mfa/verify-enroll with the first 6-digit code. Service verifies,
//     marks enrolled_at = now(), and generates 10 single-use recovery codes
//     which are returned ONCE (and never again).
//  3. On every subsequent login, if MFASecret.EnrolledAt is non-nil, the
//     login handler returns {mfa_required: true, challenge_token: <jwt>}
//     instead of the session token. The challenge token is short-lived
//     (5 min) and can only be exchanged for a real session via
//     /auth/mfa/verify with a valid TOTP code or recovery code.
package mfa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/yourorg/clickup/internal/domain"
)

var (
	ErrNotEnrolled    = errors.New("mfa: not enrolled")
	ErrInvalidCode    = errors.New("mfa: invalid code")
	ErrAlreadyEnabled = errors.New("mfa: already enrolled")
)

// Issuer is the label shown in the user's authenticator app. Set this from
// config when the service is constructed so SaaS-mode tenants see their own
// brand instead of the generic clone name.
type Service struct {
	repo   domain.MFARepo
	users  domain.UserRepo
	issuer string
}

func New(repo domain.MFARepo, users domain.UserRepo, issuer string) *Service {
	if issuer == "" {
		issuer = "ClickUp Clone"
	}
	return &Service{repo: repo, users: users, issuer: issuer}
}

type EnrollResult struct {
	// Secret is base32-encoded — the canonical RFC 6238 representation.
	// Authenticator apps that don't scan QR codes can be set up by typing
	// it in.
	Secret string `json:"secret"`
	// OtpauthURL is the otpauth://totp/... URL the frontend renders as a
	// QR code.
	OtpauthURL string `json:"otpauth_url"`
}

type VerifyEnrollResult struct {
	// RecoveryCodes are the plaintext codes shown ONCE to the user. After
	// this response we only ever store sha256(code).
	RecoveryCodes []string `json:"recovery_codes"`
}

// IsEnrolled reports whether the user has completed the enrolment flow.
// Login handlers call this to decide whether to short-circuit into the MFA
// challenge state.
func (s *Service) IsEnrolled(ctx context.Context, userID uuid.UUID) (bool, error) {
	sec, err := s.repo.GetSecret(ctx, userID)
	if err != nil || sec == nil {
		return false, err
	}
	return sec.EnrolledAt != nil, nil
}

// Enroll generates a fresh TOTP secret and persists it as un-enrolled.
// Calling this when the user is already enrolled is allowed — it resets
// the state so the user can re-pair a lost device. The recovery codes are
// invalidated as a side effect (handled by VerifyEnrollment).
func (s *Service) Enroll(ctx context.Context, userID uuid.UUID) (*EnrollResult, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrNotEnrolled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: u.Email,
		SecretSize:  20, // RFC 4226 recommends >=128 bits = 16 bytes; 20 is comfortable
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.repo.UpsertSecret(ctx, &domain.MFASecret{
		UserID:     userID,
		Secret:     key.Secret(),
		EnrolledAt: nil,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		return nil, err
	}
	// Wipe any existing recovery codes from a previous enrolment — they
	// were tied to the old secret and shouldn't unlock the new one.
	if err := s.repo.DeleteRecoveryCodes(ctx, userID); err != nil {
		return nil, err
	}

	return &EnrollResult{
		Secret:     key.Secret(),
		OtpauthURL: key.URL(),
	}, nil
}

// VerifyEnrollment finalises enrolment: it verifies the user can produce a
// valid TOTP code (proving the secret survived the QR scan), marks the
// secret as enrolled, and generates 10 recovery codes.
func (s *Service) VerifyEnrollment(ctx context.Context, userID uuid.UUID, code string) (*VerifyEnrollResult, error) {
	sec, err := s.repo.GetSecret(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sec == nil {
		return nil, ErrNotEnrolled
	}
	if sec.EnrolledAt != nil {
		return nil, ErrAlreadyEnabled
	}
	if !totp.Validate(code, sec.Secret) {
		return nil, ErrInvalidCode
	}

	now := time.Now().UTC()
	if err := s.repo.MarkEnrolled(ctx, userID, now); err != nil {
		return nil, err
	}

	codes, hashes, err := generateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}
	if err := s.repo.InsertRecoveryCodes(ctx, userID, hashes); err != nil {
		return nil, err
	}
	return &VerifyEnrollResult{RecoveryCodes: codes}, nil
}

// VerifyChallenge checks a TOTP or recovery code against an enrolled user
// during the login challenge step. It tries TOTP first (cheap, doesn't
// touch the DB beyond a single read), then falls back to recovery codes.
// On success it bumps last_used_at so the audit trail can show the most
// recent successful 2FA event.
func (s *Service) VerifyChallenge(ctx context.Context, userID uuid.UUID, code string) error {
	sec, err := s.repo.GetSecret(ctx, userID)
	if err != nil {
		return err
	}
	if sec == nil || sec.EnrolledAt == nil {
		return ErrNotEnrolled
	}
	code = strings.TrimSpace(code)

	if totp.Validate(code, sec.Secret) {
		_ = s.repo.MarkLastUsed(ctx, userID, time.Now().UTC())
		return nil
	}

	// Recovery-code path. Hash the user-supplied plaintext, then try to
	// atomically consume the matching DB row. If consumed == false the
	// code didn't exist or was already used.
	h := hashCode(code)
	consumed, err := s.repo.ConsumeRecoveryCode(ctx, userID, h)
	if err != nil {
		return err
	}
	if !consumed {
		return ErrInvalidCode
	}
	_ = s.repo.MarkLastUsed(ctx, userID, time.Now().UTC())
	return nil
}

// Disable wipes the secret and all recovery codes. Called from the user's
// settings page when they intentionally turn MFA off; returns ErrNotEnrolled
// if MFA wasn't on. Auditing is the caller's responsibility.
func (s *Service) Disable(ctx context.Context, userID uuid.UUID) error {
	sec, err := s.repo.GetSecret(ctx, userID)
	if err != nil {
		return err
	}
	if sec == nil {
		return ErrNotEnrolled
	}
	if err := s.repo.DeleteRecoveryCodes(ctx, userID); err != nil {
		return err
	}
	return s.repo.DeleteSecret(ctx, userID)
}

// RegenerateRecoveryCodes wipes existing codes and issues 10 new ones. The
// new plaintext set is returned exactly once; the user is expected to write
// them down.
func (s *Service) RegenerateRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	sec, err := s.repo.GetSecret(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sec == nil || sec.EnrolledAt == nil {
		return nil, ErrNotEnrolled
	}
	if err := s.repo.DeleteRecoveryCodes(ctx, userID); err != nil {
		return nil, err
	}
	codes, hashes, err := generateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}
	if err := s.repo.InsertRecoveryCodes(ctx, userID, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}

func (s *Service) RemainingRecoveryCodes(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountRemainingRecoveryCodes(ctx, userID)
}

// generateRecoveryCodes returns N plaintext codes (base32, 10 chars each —
// 50 bits of entropy) and their sha256 hashes. The plaintext is returned to
// the caller; only the hash is ever persisted.
func generateRecoveryCodes(n int) (plain []string, hashes [][]byte, err error) {
	plain = make([]string, n)
	hashes = make([][]byte, n)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	for i := 0; i < n; i++ {
		raw := make([]byte, 10) // 80 random bits → 16 base32 chars; we slice to 10
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, err
		}
		s := enc.EncodeToString(raw)
		if len(s) > 10 {
			s = s[:10]
		}
		// Format with a hyphen for readability: "ABCDE-FGHIJ"
		formatted := strings.ToUpper(s[:5] + "-" + s[5:])
		plain[i] = formatted
		hashes[i] = hashCode(formatted)
	}
	return plain, hashes, nil
}

// hashCode normalises and hashes a recovery code. Normalisation strips
// formatting (whitespace, hyphens, case) so the user can paste it from a
// password manager without worrying about exact form.
func hashCode(code string) []byte {
	norm := strings.ToUpper(strings.NewReplacer(" ", "", "-", "", "\t", "").Replace(code))
	sum := sha256.Sum256([]byte(norm))
	return sum[:]
}

// HashCodeHex is an external helper for tests / debugging that need to
// reproduce hashes deterministically. Not used by production code.
func HashCodeHex(code string) string {
	return hex.EncodeToString(hashCode(code))
}
