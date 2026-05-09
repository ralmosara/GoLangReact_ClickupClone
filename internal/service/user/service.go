package user

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/middleware"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
	ErrMFARequired        = errors.New("mfa code required")
	ErrInvalidMFACode     = errors.New("invalid mfa code")
	ErrEmailRequired      = errors.New("email is required")
)

// MFAChecker is the slice of the MFA service the user service depends on.
// Defining it here (rather than importing service/mfa) prevents an import
// cycle if mfa later wants to record audit events through a user-side
// hook. nil is allowed — when MFAChecker is nil, MFA is treated as
// universally disabled.
type MFAChecker interface {
	IsEnrolled(ctx context.Context, userID uuid.UUID) (bool, error)
	VerifyChallenge(ctx context.Context, userID uuid.UUID, code string) error
}

// SessionIssuer is the slice of the session service this package needs to
// persist a row alongside each freshly-minted token. Optional — when nil,
// tokens are still valid (the auth middleware tolerates absent sessions
// for backwards compatibility) but they can't be revoked from the
// Active Sessions UI.
type SessionIssuer interface {
	Issue(ctx context.Context, in SessionIssueInput) error
}

// SessionIssueInput is the contract between user.Service and the session
// service — the user service has the jti + expiry; the handler layer
// passes UA/IP via context (set by middleware.SessionContext below).
type SessionIssueInput struct {
	JTI       uuid.UUID
	UserID    uuid.UUID
	UserAgent string
	IP        string
	ExpiresAt time.Time
}

type Service struct {
	users        domain.UserRepo
	jwtSecret    string
	tokenTTL     time.Duration
	challengeTTL time.Duration
	mfa          MFAChecker
	sessions     SessionIssuer
}

func New(users domain.UserRepo, jwtSecret string) *Service {
	return &Service{
		users:        users,
		jwtSecret:    jwtSecret,
		tokenTTL:     7 * 24 * time.Hour,
		challengeTTL: 5 * time.Minute,
	}
}

// WithMFA returns the same service wired up with an MFA checker. main.go
// calls this once mfa service is constructed; calling it twice is fine
// (the second value wins).
func (s *Service) WithMFA(mfa MFAChecker) *Service {
	s.mfa = mfa
	return s
}

// WithSessions wires the session-issuing dependency. Same shape as
// WithMFA — chainable from main.go, no-op when called with nil.
func (s *Service) WithSessions(issuer SessionIssuer) *Service {
	s.sessions = issuer
	return s
}

// LoginContext carries the per-request bits (UA, IP) that need to land
// in the session row. The handler builds this from r.Header / r.RemoteAddr
// before calling Login; legacy callers can pass a zero value and the
// session row will just have empty UA/IP.
type LoginContext struct {
	UserAgent string
	IP        string
}

// AdminCreateInput is the payload for admin-driven user creation. Mirrors
// the old self-registration shape. Kept under this name so the seeder
// and the member service share one type.
type AdminCreateInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResult is the shape returned by Register, Login, and VerifyMFAChallenge.
//
// When MFARequired is true, only ChallengeToken is populated — Token and
// User are left empty. Clients are expected to:
//   - if MFARequired == false: store Token and proceed.
//   - if MFARequired == true:  collect a TOTP/recovery code from the user
//     and POST it to /auth/mfa/verify with ChallengeToken.
type AuthResult struct {
	Token          string       `json:"token,omitempty"`
	User           *domain.User `json:"user,omitempty"`
	MFARequired    bool         `json:"mfa_required,omitempty"`
	ChallengeToken string       `json:"challenge_token,omitempty"`
}

// AdminCreate creates a new user from an authenticated admin path
// (workspace member-invite, OIDC JIT, seeder). It does NOT issue a JWT —
// callers are expected to either return the user's id to the admin or
// route the user through /auth/login afterwards.
//
// This used to be the public Register() handler; self-registration was
// removed in favour of admin-driven creation, but the underlying flow
// (validate → bcrypt → users.Create) is unchanged.
func (s *Service) AdminCreate(ctx context.Context, in AdminCreateInput) (*domain.User, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Email == "" {
		return nil, ErrEmailRequired
	}
	if len(in.Password) < 8 {
		return nil, ErrWeakPassword
	}
	existing, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &domain.User{Email: in.Email, PasswordHash: string(hash), Name: in.Name}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	return s.LoginWithContext(ctx, in, LoginContext{})
}

// LoginWithContext is the request-aware variant — handlers should prefer
// this over Login() so the session row can attribute UA/IP correctly.
// Login() is kept as a thin wrapper for any caller that doesn't have an
// HTTP request to source UA/IP from (the seeder, mainly).
func (s *Service) LoginWithContext(ctx context.Context, in LoginInput, lc LoginContext) (*AuthResult, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	u, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// If the user has MFA enrolled, short-circuit to the challenge state —
	// password alone never grants a session token.
	if s.mfa != nil {
		enrolled, err := s.mfa.IsEnrolled(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		if enrolled {
			challenge, err := middleware.IssueChallengeToken(s.jwtSecret, u.ID, s.challengeTTL)
			if err != nil {
				return nil, err
			}
			return &AuthResult{MFARequired: true, ChallengeToken: challenge}, nil
		}
	}

	return s.mintSession(ctx, u, lc)
}

// VerifyMFAChallenge is the second leg of an MFA login. The challenge
// token proves the password check succeeded; the code proves the second
// factor. Both must be valid to mint a session token.
func (s *Service) VerifyMFAChallenge(ctx context.Context, challengeToken, code string) (*AuthResult, error) {
	return s.VerifyMFAChallengeWithContext(ctx, challengeToken, code, LoginContext{})
}

// VerifyMFAChallengeWithContext mirrors LoginWithContext — request-aware
// variant that lets the session row capture UA/IP.
func (s *Service) VerifyMFAChallengeWithContext(ctx context.Context, challengeToken, code string, lc LoginContext) (*AuthResult, error) {
	if s.mfa == nil {
		return nil, ErrInvalidCredentials
	}
	uid, err := middleware.ParseChallengeToken(s.jwtSecret, challengeToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := s.mfa.VerifyChallenge(ctx, uid, code); err != nil {
		return nil, ErrInvalidMFACode
	}
	u, err := s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrInvalidCredentials
	}
	return s.mintSession(ctx, u, lc)
}

// mintSession is the shared tail of Login and VerifyMFAChallenge: issue
// a JWT, persist a session row keyed by the JWT's jti (best-effort —
// session persistence failure does NOT prevent login, since the auth
// middleware tolerates missing sessions for older tokens), and return
// the AuthResult.
func (s *Service) mintSession(ctx context.Context, u *domain.User, lc LoginContext) (*AuthResult, error) {
	token, jti, err := middleware.IssueToken(s.jwtSecret, u.ID, s.tokenTTL)
	if err != nil {
		return nil, err
	}
	if s.sessions != nil {
		// Failure here is logged-and-ignored — we don't want a session
		// table outage to lock everyone out. The token will still
		// validate (sessions are tolerated as absent by SessionGate),
		// the user just won't appear in their own Active Sessions
		// list until next login when persistence recovers.
		if err := s.sessions.Issue(ctx, SessionIssueInput{
			JTI:       jti,
			UserID:    u.ID,
			UserAgent: lc.UserAgent,
			IP:        lc.IP,
			ExpiresAt: time.Now().Add(s.tokenTTL),
		}); err != nil {
			// Most common cause: migration 033 (user_sessions table)
			// hasn't been applied yet. Surface it via slog so the
			// operator notices rather than wondering why their
			// Active Sessions tab is always empty.
			slog.WarnContext(ctx, "session issue failed; token still valid (JWT-only auth)",
				"err", err, "user_id", u.ID.String())
		}
	}
	return &AuthResult{Token: token, User: u}, nil
}

func (s *Service) Me(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

// ── User Management surface ─────────────────────────────────────────────
//
// These methods back the workspace-scoped User Management page. Operations
// take a workspace id so service-level helpers can enforce "target user must
// be a member of this workspace" — the policy layer enforces the calling
// user has user.manage on that workspace.

// ErrUserNotMember is returned when the caller tries to manage a user who
// isn't a member of the workspace they have user.manage on. Cross-tenant
// management is intentionally forbidden in this iteration.
var ErrUserNotMember = errors.New("user is not a member of this workspace")

// ListWorkspaceUsers returns every user in the workspace with their role
// and joined-at timestamp.
func (s *Service) ListWorkspaceUsers(ctx context.Context, workspaceID uuid.UUID) ([]domain.WorkspaceUser, error) {
	return s.users.ListByWorkspace(ctx, workspaceID)
}

// UpdateName edits the user's display name. Caller must have established
// (via the policy layer + the membership gate) that they have user.manage
// on a workspace the target user belongs to.
func (s *Service) UpdateName(ctx context.Context, id uuid.UUID, name string) (*domain.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	if err := s.users.UpdateName(ctx, id, name); err != nil {
		return nil, err
	}
	return s.users.GetByID(ctx, id)
}

// ResetPassword takes the new plaintext password, validates length, bcrypt-
// hashes it, and writes it. The current password is NOT required — this is
// an admin reset flow, not a user self-service flow.
func (s *Service) ResetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.users.UpdatePasswordHash(ctx, id, string(hash))
}

// ChangeOwnPassword is the user-self path: requires the current password
// before accepting the new one. Surface errors are mapped by the handler;
// here we keep the bcrypt-failure → ErrInvalidCredentials mapping local
// so the timing of "user not found" vs "wrong password" doesn't leak.
func (s *Service) ChangeOwnPassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.users.UpdatePasswordHash(ctx, userID, string(hash))
}

// Delete hard-deletes the user. The cascade is described on
// repository/user/repository.go's Delete; in short, FK ON DELETE CASCADE
// removes membership/MFA/recovery-codes/OIDC-identities, and ON DELETE
// SET NULL keeps history rows (comments, audit log, attachments) intact
// with a null actor.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.users.Delete(ctx, id)
}

// ── Global admin surface ────────────────────────────────────────────────

// ListAll returns a single page of every user in the system, plus the
// total count so the UI can render pagination. The caller has already
// passed the user.manage-anywhere gate at the handler layer.
func (s *Service) ListAll(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
	users, err := s.users.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.users.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

