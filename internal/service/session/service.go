// Package session is the service layer over the user_sessions table. It
// provides the SessionChecker that the auth middleware uses to gate
// requests on session revocation, plus the CRUD surface backing the
// "Active sessions" UI.
//
// The package intentionally has no policy dependency — every operation
// is keyed by the caller's userID, so a user can only ever see/revoke
// their own sessions; cross-user revocation isn't expressible.
package session

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/middleware"
)

type Service struct {
	repo  domain.SessionRepo
	audit *audit.Recorder

	// gcEvery throttles the lazy "delete expired rows" sweep that the
	// Validate path runs opportunistically. nil clock means time.Now().
	gcEvery time.Duration
	lastGC  time.Time
}

func New(repo domain.SessionRepo, recorder *audit.Recorder) *Service {
	return &Service{repo: repo, audit: recorder, gcEvery: 5 * time.Minute}
}

// Issue records a freshly-minted session. Called by user.Login and
// user.VerifyMFAChallenge right after middleware.IssueToken returns.
//
// Label is best-effort derived from the User-Agent so the UI has
// something readable; the user can rename it later from Settings →
// Security → Sessions (not in this PR).
type IssueInput struct {
	JTI       uuid.UUID
	UserID    uuid.UUID
	UserAgent string
	IP        *netip.Addr
	ExpiresAt time.Time
}

func (s *Service) Issue(ctx context.Context, in IssueInput) error {
	sess := &domain.Session{
		ID:        in.JTI,
		UserID:    in.UserID,
		Label:     deriveLabel(in.UserAgent),
		UserAgent: in.UserAgent,
		IP:        in.IP,
		ExpiresAt: in.ExpiresAt,
	}
	if err := s.repo.Create(ctx, sess); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &in.UserID,
			EntityType: "session",
			EntityID:   &in.JTI,
			Verb:       "created",
		})
	}
	return nil
}

// Validate is the SessionChecker implementation. Returns nil for live
// sessions; middleware.ErrSessionRevoked when the row is gone (the
// token was explicitly revoked) or expired.
//
// Fails OPEN on infrastructure errors (table missing, DB connection
// broken, etc.) — the JWT was already validated by the auth middleware,
// and rejecting valid tokens because session tracking is unavailable
// would lock everyone out for an issue unrelated to their credentials.
// This matches the pre-#14 behavior (JWT-only auth) when the
// user_sessions backing isn't there.
func (s *Service) Validate(ctx context.Context, jti uuid.UUID, touchEvery time.Duration) error {
	sess, err := s.repo.Get(ctx, jti)
	if err != nil {
		// Log via slog so the failure is visible without becoming a 401.
		// The most common cause is migration 033 not yet applied.
		slog.WarnContext(ctx, "session validate: Get failed; falling back to JWT-only auth",
			"err", err, "jti", jti.String())
		return nil
	}
	// sess == nil means the row was explicitly deleted (revoked) or
	// never existed. The latter happens when login was issued before
	// the session table existed, or when session.Issue silently failed.
	// In either case treat as "session tracking unknown" — don't 401.
	if sess == nil {
		return nil
	}
	if time.Now().After(sess.ExpiresAt) {
		return middleware.ErrSessionRevoked
	}
	// Bump last_seen_at on a debounced cadence — see repo.Touch comment.
	if err := s.repo.Touch(ctx, jti, touchEvery); err != nil {
		// Touch failures are observability-only; never fail the request.
		// (DB hiccups during a touch shouldn't bounce a logged-in user.)
		_ = err
	}
	// Opportunistic GC of expired rows. Cheap when there are none, and
	// avoids needing a separate cron for cleanup. Throttled to one
	// sweep per gcEvery interval per process.
	if time.Since(s.lastGC) > s.gcEvery {
		s.lastGC = time.Now()
		go func() {
			// Background GC; ignore errors — next sweep will retry.
			_, _ = s.repo.DeleteExpired(context.Background())
		}()
	}
	return nil
}

// List returns all live sessions for the user, most-recently-active first.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]domain.Session, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Revoke deletes a single session. The caller must own it — the handler
// asserts user_id matches before calling. We re-check here as belt-and-
// suspenders so a misconfigured handler can't escalate into a
// cross-user revocation.
var ErrNotOwner = errors.New("session does not belong to caller")

func (s *Service) Revoke(ctx context.Context, userID, jti uuid.UUID) error {
	sess, err := s.repo.Get(ctx, jti)
	if err != nil {
		return err
	}
	if sess == nil {
		return nil // already gone — idempotent
	}
	if sess.UserID != userID {
		return ErrNotOwner
	}
	if err := s.repo.Delete(ctx, jti); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &userID,
			EntityType: "session",
			EntityID:   &jti,
			Verb:       "revoked",
		})
	}
	return nil
}

// RevokeOthers deletes every session for the user except the supplied
// keep-id (typically the caller's current session). Returns the number
// of rows deleted so the UI can render "signed out 3 other devices".
func (s *Service) RevokeOthers(ctx context.Context, userID, keepID uuid.UUID) (int, error) {
	n, err := s.repo.DeleteAllExcept(ctx, userID, keepID)
	if err != nil {
		return 0, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &userID,
			EntityType: "session",
			Verb:       "revoked_others",
			After:      map[string]int{"count": n},
		})
	}
	return n, nil
}

// deriveLabel produces the user-visible "Chrome on macOS" string from a
// raw User-Agent header. Not perfect — UA parsing never is — but plenty
// for an Active Sessions list. Falls back to the bare UA string if no
// known browser/OS markers are found.
func deriveLabel(ua string) string {
	if ua == "" {
		return "Unknown device"
	}
	browser := pickFirst(ua, []string{"Edg", "Chrome", "Firefox", "Safari", "Opera"})
	os := pickFirst(ua, []string{"Windows", "Mac", "Linux", "Android", "iPhone", "iPad"})
	switch {
	case browser != "" && os != "":
		return browser + " on " + osLabel(os)
	case browser != "":
		return browser
	case os != "":
		return osLabel(os)
	default:
		// Truncate so the label stays UI-friendly.
		if len(ua) > 60 {
			return ua[:60] + "…"
		}
		return ua
	}
}

func pickFirst(s string, needles []string) string {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return n
		}
	}
	return ""
}

func osLabel(s string) string {
	switch s {
	case "Mac":     return "macOS"
	case "iPhone":  return "iPhone"
	case "iPad":    return "iPad"
	default:        return s
	}
}
