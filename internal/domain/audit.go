package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditEntry struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID *uuid.UUID      `json:"workspace_id,omitempty"`
	ActorID     *uuid.UUID      `json:"actor_id,omitempty"`
	EntityType  string          `json:"entity_type"`
	EntityID    *uuid.UUID      `json:"entity_id,omitempty"`
	Verb        string          `json:"verb"`
	BeforeJSON  json.RawMessage `json:"before,omitempty"`
	AfterJSON   json.RawMessage `json:"after,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type AuditFilter struct {
	WorkspaceID *uuid.UUID
	EntityType  *string
	EntityID    *uuid.UUID
	// ActorID restricts to a single actor's actions — backs the "show only
	// what alice did" filter on the activity log UI.
	ActorID *uuid.UUID
	// Verb restricts to a verb (created/updated/deleted/...) — handy for
	// "show me every delete in the last 30 days" investigations.
	Verb *string
	// Since is the lower bound on created_at (inclusive). Pairs with
	// Before to bracket the time window. Both nil = unbounded.
	Since  *time.Time
	Limit  int
	Before *time.Time
}

type AuditRepo interface {
	Create(ctx context.Context, e *AuditEntry) error
	List(ctx context.Context, f AuditFilter) ([]AuditEntry, error)
}
