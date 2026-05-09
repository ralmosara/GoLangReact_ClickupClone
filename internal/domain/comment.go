package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID              uuid.UUID  `json:"id"`
	TaskID          uuid.UUID  `json:"task_id"`
	AuthorID        *uuid.UUID `json:"author_id,omitempty"`
	// ParentCommentID is non-nil when this comment is a reply.
	// Top-level comments leave it null. Single-level threading is
	// enforced at the service layer.
	ParentCommentID *uuid.UUID `json:"parent_comment_id,omitempty"`
	Body            string     `json:"body"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	// EditedAt is set only when the body has been edited at least
	// once. The UI uses presence to render the "(edited)" tag.
	EditedAt        *time.Time `json:"edited_at,omitempty"`
	// DeletedAt+DeletedBy mark soft-deleted rows. Soft-delete keeps
	// thread structure intact ("[Comment removed]") so replies don't
	// orphan when a parent is removed.
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	DeletedBy       *uuid.UUID `json:"deleted_by,omitempty"`

	// ReplyCount is populated on top-level comments only — a denormalised
	// SELECT COUNT(*) WHERE parent_comment_id = id. Lets the UI render
	// "3 replies" without N+1.
	ReplyCount int `json:"reply_count"`

	// Reactions is the per-emoji aggregate. Populated on listing only —
	// individual reaction rows aren't shipped over the wire because the
	// only thing the UI needs is the count plus a flag for "did I react".
	Reactions []CommentReactionSummary `json:"reactions,omitempty"`
}

// CommentReactionSummary is one emoji bucket: the emoji string itself
// (e.g. "👍"), the total count, and whether the calling user is one of
// the reactors. Per-user reactor lists aren't shipped — they'd 10x the
// payload on a busy thread without giving the UI much.
type CommentReactionSummary struct {
	Emoji   string `json:"emoji"`
	Count   int    `json:"count"`
	Reacted bool   `json:"reacted"`
}

type CommentRepo interface {
	Create(ctx context.Context, c *Comment) error
	// ListByTask returns top-level comments for a task. Replies are
	// fetched per-parent via ListReplies — the UI lazily expands a
	// thread on click. ReplyCount is populated on each row.
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]Comment, error)
	// ListReplies returns the children of one parent comment ordered
	// oldest-first to read like a chat thread.
	ListReplies(ctx context.Context, parentID uuid.UUID) ([]Comment, error)
	// GetByID fetches one comment row, used by Update / Delete to gate
	// "author owns it" before mutating.
	GetByID(ctx context.Context, id uuid.UUID) (*Comment, error)
	// UpdateBody replaces the body and stamps edited_at = now().
	// Returns the updated row.
	UpdateBody(ctx context.Context, id uuid.UUID, body string) (*Comment, error)
	// Delete is the legacy hard-delete; SoftDelete is preferred when the
	// caller wants thread structure preserved.
	Delete(ctx context.Context, id uuid.UUID) error
	// SoftDelete marks the row deleted_at = now() and records the actor.
	// Body is left untouched (the UI replaces it client-side).
	SoftDelete(ctx context.Context, id, actor uuid.UUID) error

	// Reactions ----------------------------------------------------------

	// AddReaction is idempotent — re-reacting with the same emoji is a
	// no-op (PRIMARY KEY conflict ON CONFLICT DO NOTHING).
	AddReaction(ctx context.Context, commentID, userID uuid.UUID, emoji string) error
	RemoveReaction(ctx context.Context, commentID, userID uuid.UUID, emoji string) error
	// ListReactionSummaries returns per-emoji counts for the supplied
	// comments in a single round-trip — used to populate the Reactions
	// field on a list response. Map key is comment_id.
	ListReactionSummaries(ctx context.Context, commentIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]CommentReactionSummary, error)
}
