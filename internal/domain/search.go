package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SearchHit is a cross-entity result row. EntityType identifies which entity
// the hit came from (task / doc / comment / message); the UI uses it to pick
// an icon and link target.
type SearchHit struct {
	EntityType string     `json:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id"`
	Title      string     `json:"title"`
	Snippet    string     `json:"snippet"`
	Rank       float64    `json:"rank"`
	// Context pointers — any that apply for the entity type.
	WorkspaceID *uuid.UUID `json:"workspace_id,omitempty"`
	ListID      *uuid.UUID `json:"list_id,omitempty"`
	TaskID      *uuid.UUID `json:"task_id,omitempty"`
	DocID       *uuid.UUID `json:"doc_id,omitempty"`
	ChannelID   *uuid.UUID `json:"channel_id,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

type SearchRepo interface {
	Search(ctx context.Context, workspaceID uuid.UUID, query string, limit int) ([]SearchHit, error)
}
