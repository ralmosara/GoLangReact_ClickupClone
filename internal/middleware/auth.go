package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ctxKey string

const (
	userIDKey  ctxKey = "userID"
	sessionIDKey ctxKey = "sessionID"
)

// Token scopes. A blank scope is treated as ScopeSession for backwards
// compatibility with tokens issued before MFA was added.
const (
	ScopeSession      = "session"
	ScopeMFAChallenge = "mfa_challenge"
)

type Claims struct {
	UserID string `json:"sub"`
	Scope  string `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

// ErrInvalidToken is returned by ParseToken when the JWT is missing, malformed,
// or signed with a different secret.
var ErrInvalidToken = errors.New("invalid token")

// IssueToken issues a session token. Existing callers pre-MFA continue to
// work — the scope is stamped here so the auth middleware can reject
// challenge tokens cleanly. The returned jti is the session id; callers
// that track sessions in the DB persist it under that key so the auth
// middleware can resolve it back to a Session row.
func IssueToken(secret string, userID uuid.UUID, ttl time.Duration) (string, uuid.UUID, error) {
	jti := uuid.New()
	tok, err := issueScopedToken(secret, userID, ttl, ScopeSession, jti)
	return tok, jti, err
}

// IssueChallengeToken issues a short-lived token that proves the holder
// passed password authentication. It can be exchanged for a real session
// token via /auth/mfa/verify but is rejected by the regular Auth middleware
// — so it can't be used to call any other API. Challenge tokens carry no
// jti since they're not session-bearing.
func IssueChallengeToken(secret string, userID uuid.UUID, ttl time.Duration) (string, error) {
	tok, err := issueScopedToken(secret, userID, ttl, ScopeMFAChallenge, uuid.Nil)
	return tok, err
}

func issueScopedToken(secret string, userID uuid.UUID, ttl time.Duration, scope string, jti uuid.UUID) (string, error) {
	reg := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
	}
	if jti != uuid.Nil {
		reg.ID = jti.String()
	}
	claims := Claims{
		UserID:           userID.String(),
		Scope:            scope,
		RegisteredClaims: reg,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseToken verifies a session-scoped JWT and returns the user id.
// Tokens with a non-session scope (e.g. an MFA challenge token) are
// rejected — they can only be parsed via ParseChallengeToken. Existing
// callers that don't need the jti can keep using this; new callers that
// want to gate on session revocation should use ParseTokenWithJTI.
func ParseToken(secret, tokenStr string) (uuid.UUID, error) {
	uid, _, err := ParseTokenWithJTI(secret, tokenStr)
	return uid, err
}

// ParseTokenWithJTI is the session-aware variant of ParseToken — also
// returns the JWT's jti so the caller can look up a Session row.
// Returns uuid.Nil for the jti on tokens minted before sessions
// existed (i.e. without an `id` claim).
func ParseTokenWithJTI(secret, tokenStr string) (uuid.UUID, uuid.UUID, error) {
	uid, scope, jti, err := parseAnyScope(secret, tokenStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if scope != "" && scope != ScopeSession {
		return uuid.Nil, uuid.Nil, ErrInvalidToken
	}
	return uid, jti, nil
}

// ParseChallengeToken verifies a token issued by IssueChallengeToken and
// returns the userID it was minted for. Used by /auth/mfa/verify.
func ParseChallengeToken(secret, tokenStr string) (uuid.UUID, error) {
	uid, scope, _, err := parseAnyScope(secret, tokenStr)
	if err != nil {
		return uuid.Nil, err
	}
	if scope != ScopeMFAChallenge {
		return uuid.Nil, ErrInvalidToken
	}
	return uid, nil
}

func parseAnyScope(secret, tokenStr string) (uuid.UUID, string, uuid.UUID, error) {
	if tokenStr == "" {
		return uuid.Nil, "", uuid.Nil, ErrInvalidToken
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, "", uuid.Nil, ErrInvalidToken
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, "", uuid.Nil, ErrInvalidToken
	}
	var jti uuid.UUID
	if claims.ID != "" {
		// Tolerate malformed jti — treat as "no session tracking" rather
		// than rejecting the whole token. The session middleware can
		// still reject tokens with an empty jti when it cares.
		if parsed, err := uuid.Parse(claims.ID); err == nil {
			jti = parsed
		}
	}
	return uid, claims.Scope, jti, nil
}

// TokenFromRequest pulls the JWT from either the `Authorization: Bearer` header
// or a `token` query parameter. The query-param path exists so browser
// WebSocket clients — which can't set custom headers — can still authenticate.
func TokenFromRequest(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeErr(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			uid, jti, err := ParseTokenWithJTI(secret, strings.TrimPrefix(auth, "Bearer "))
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, uid)
			if jti != uuid.Nil {
				ctx = context.WithValue(ctx, sessionIDKey, jti)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SessionGate is the optional, session-revocation-aware layer that goes
// directly *after* Auth. It looks up the jti from the token and rejects
// the request if the matching session row has been revoked (deleted) or
// expired. When a checker isn't wired (cfg.Sessions == nil), it's a no-op,
// so this middleware is safe to mount unconditionally.
//
// Touching last_seen_at is debounced via the repo — a request only writes
// when the row's last_seen_at is older than touchEvery. Set to one minute
// it bounds DB write rate to 1/min/device regardless of request rate.
type SessionChecker interface {
	// Validate returns nil when the session is live. ErrSessionRevoked
	// when the row is gone or expired. Any other error is a downstream
	// failure (DB down) — the middleware fails-closed (401) rather than
	// silently let the request through.
	Validate(ctx context.Context, jti uuid.UUID, touchEvery time.Duration) error
}

// ErrSessionRevoked is returned by SessionChecker.Validate when the jti
// has been deleted or has passed its exp.
var ErrSessionRevoked = errors.New("session revoked")

func SessionGate(checker SessionChecker, touchEvery time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if checker == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jti, ok := SessionID(r.Context())
			if !ok {
				// Token didn't carry a jti — old token from before
				// sessions existed. Allow through; it'll naturally
				// expire. New tokens always carry one.
				next.ServeHTTP(w, r)
				return
			}
			if err := checker.Validate(r.Context(), jti, touchEvery); err != nil {
				writeErr(w, http.StatusUnauthorized, "session revoked")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AttachUserID lets non-middleware callers (e.g. the WS handler) set the id on
// a derived context the same way Auth does.
func AttachUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// AttachSessionID is the symmetric helper for the jti — used by paths
// that mint a token themselves (login, mfa-verify) when they want the
// downstream chain to see it.
func AttachSessionID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, sessionIDKey, id)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(userIDKey).(uuid.UUID)
	return v, ok
}

// SessionID returns the JWT's jti from the request context. Present on
// any request authenticated by the Auth middleware whose token was minted
// with one. Absent on legacy tokens — callers should treat absent as
// "session-tracking not available" rather than as an error.
func SessionID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(sessionIDKey).(uuid.UUID)
	return v, ok
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
