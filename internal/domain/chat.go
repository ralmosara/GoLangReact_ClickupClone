package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	ChannelKindChannel = "channel"
	ChannelKindDM      = "dm"
)

type Channel struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	SpaceID     *uuid.UUID `json:"space_id,omitempty"`
	Name        string     `json:"name"`
	Topic       string     `json:"topic"`
	Kind        string     `json:"kind"`
	IsPrivate   bool       `json:"is_private"`
	CreatorID   *uuid.UUID `json:"creator_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Message struct {
	ID              uuid.UUID  `json:"id"`
	ChannelID       uuid.UUID  `json:"channel_id"`
	AuthorID        *uuid.UUID `json:"author_id,omitempty"`
	ParentMessageID *uuid.UUID `json:"parent_message_id,omitempty"`
	Body            string     `json:"body"`
	EditedAt        *time.Time `json:"edited_at,omitempty"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ChannelRepo interface {
	Create(ctx context.Context, c *Channel) error
	GetByID(ctx context.Context, id uuid.UUID) (*Channel, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Channel, error)
	ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]Channel, error)
	Update(ctx context.Context, c *Channel) error
	Delete(ctx context.Context, id uuid.UUID) error

	AddMember(ctx context.Context, channelID, userID uuid.UUID) error
	RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error
	ListMembers(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error)
	TouchRead(ctx context.Context, channelID, userID uuid.UUID) error
}

type MessageFilter struct {
	ChannelID uuid.UUID
	ParentID  *uuid.UUID
	Before    *time.Time
	Limit     int
}

type MessageRepo interface {
	Create(ctx context.Context, m *Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*Message, error)
	List(ctx context.Context, f MessageFilter) ([]Message, error)
	Update(ctx context.Context, m *Message) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	CountThreadReplies(ctx context.Context, parentID uuid.UUID) (int, error)
}
