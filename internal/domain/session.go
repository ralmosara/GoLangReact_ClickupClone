package domain

import (
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// Session is one active device/browser tab whose JWT is still trusted.
// Created on /auth/login, deleted by the user from the Active Sessions UI
// (or implicitly when the JWT exp passes — the middleware GCs lazily).
//
// JSON shape is what the FE consumes; never carry the raw token here.
type Session struct {
	ID         uuid.UUID    `json:"id"`         // jti — also the FK from the JWT itself
	UserID     uuid.UUID    `json:"user_id"`
	Label      string       `json:"label"`      // user-visible "Chrome on macOS"
	UserAgent  string       `json:"user_agent"`
	IP         *netip.Addr  `json:"ip,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	LastSeenAt time.Time    `json:"last_seen_at"`
	ExpiresAt  time.Time    `json:"expires_at"`
}

type SessionRepo interface {
	// Create inserts a new session row keyed by the JWT's jti.
	Create(ctx context.Context, s *Session) error

	// Get returns the session by jti, or (nil, nil) if absent. Used by
	// the auth middleware to confirm the token hasn't been revoked.
	Get(ctx context.Context, id uuid.UUID) (*Session, error)

	// ListByUser returns every non-expired session for the user, most-
	// recently-active first.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Session, error)

	// Touch bumps last_seen_at to now() — only when the existing
	// last_seen_at is older than the supplied threshold. The repo skips
	// the UPDATE entirely when the row is newer, so request-path cost
	// stays bounded (one cheap row read + an occasional write per minute
	// per device).
	Touch(ctx context.Context, id uuid.UUID, olderThan time.Duration) error

	// Delete revokes a single session.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteAllExcept revokes every session for the user EXCEPT the
	// supplied jti. Used by "sign out all other devices" — the caller's
	// current token survives so they're not bounced out of their own UI.
	DeleteAllExcept(ctx context.Context, userID, keepID uuid.UUID) (int, error)

	// DeleteExpired GCs rows past their exp. Called periodically by the
	// auth middleware so the table doesn't grow unbounded.
	DeleteExpired(ctx context.Context) (int, error)
}
