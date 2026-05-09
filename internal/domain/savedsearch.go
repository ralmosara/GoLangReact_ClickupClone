package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SavedSearch is a per-user bookmark over the search palette. Stored in
// the saved_searches table and surfaced in the palette as a list of
// "Pinned" / "Recent" entries the user can re-run with one click.
type SavedSearch struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	WorkspaceID  uuid.UUID       `json:"workspace_id"`
	Name         string          `json:"name"`
	EntityType   *string         `json:"entity_type,omitempty"`
	QueryText    string          `json:"query_text"`
	Filters      json.RawMessage `json:"filters"`
	Pinned       bool            `json:"pinned"`
	LastUsedAt   *time.Time      `json:"last_used_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type SavedSearchRepo interface {
	Create(ctx context.Context, s *SavedSearch) error
	GetByID(ctx context.Context, id uuid.UUID) (*SavedSearch, error)
	ListForUser(ctx context.Context, userID, workspaceID uuid.UUID) ([]SavedSearch, error)
	Update(ctx context.Context, s *SavedSearch) error
	Delete(ctx context.Context, id uuid.UUID) error
	// TouchUsed bumps last_used_at = now() — called after Run so the
	// "recent" bucket stays accurate without a Read-Modify-Write.
	TouchUsed(ctx context.Context, id uuid.UUID) error
}
