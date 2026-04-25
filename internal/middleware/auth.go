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

const userIDKey ctxKey = "userID"

type Claims struct {
	UserID string `json:"sub"`
	jwt.RegisteredClaims
}

// ErrInvalidToken is returned by ParseToken when the JWT is missing, malformed,
// or signed with a different secret.
var ErrInvalidToken = errors.New("invalid token")

func IssueToken(secret string, userID uuid.UUID, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseToken verifies a JWT string and returns the user id. Shared by the HTTP
// auth middleware and the WebSocket handshake guard.
func ParseToken(secret, tokenStr string) (uuid.UUID, error) {
	if tokenStr == "" {
		return uuid.Nil, ErrInvalidToken
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return uid, nil
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
			uid, err := ParseToken(secret, strings.TrimPrefix(auth, "Bearer "))
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AttachUserID lets non-middleware callers (e.g. the WS handler) set the id on
// a derived context the same way Auth does.
func AttachUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(userIDKey).(uuid.UUID)
	return v, ok
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
