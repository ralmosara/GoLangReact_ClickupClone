package ws

import (
	"encoding/json"
	"time"
)

// Event is the v1 envelope broadcast over WebSocket. Keep field additions
// additive — consumers should tolerate unknown fields.
type Event struct {
	V           int             `json:"v"`
	Room        string          `json:"room"`
	Type        string          `json:"type"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	ActorID     string          `json:"actor_id,omitempty"`
	EntityID    string          `json:"entity_id,omitempty"`
	TS          time.Time       `json:"ts"`
	Payload     json.RawMessage `json:"payload,omitempty"`
}

// Event type constants grouped by entity.
const (
	EventTaskCreated = "task.created"
	EventTaskUpdated = "task.updated"
	EventTaskDeleted = "task.deleted"
	EventTaskMoved   = "task.moved"

	EventCommentCreated  = "comment.created"
	EventCommentUpdated  = "comment.updated"
	EventCommentDeleted  = "comment.deleted"
	EventCommentReaction = "comment.reaction"
	EventComment         = EventCommentCreated // legacy alias

	EventStatusCreated = "status.created"
	EventStatusUpdated = "status.updated"
	EventStatusDeleted = "status.deleted"

	EventTagAttached = "tag.attached"
	EventTagDetached = "tag.detached"

	EventAssigneeAdded   = "assignee.added"
	EventAssigneeRemoved = "assignee.removed"

	EventAttachmentCreated = "attachment.created"
	EventAttachmentDeleted = "attachment.deleted"

	EventNotificationCreated = "notification.created"

	EventMemberAdded   = "member.added"
	EventMemberRemoved = "member.removed"

	EventSubtaskCreated = "subtask.created"

	EventDocCreated = "doc.created"
	EventDocUpdated = "doc.updated"
	EventDocDeleted = "doc.deleted"

	EventChannelCreated = "channel.created"
	EventChannelUpdated = "channel.updated"
	EventChannelDeleted = "channel.deleted"

	EventMessageCreated = "message.created"
	EventMessageUpdated = "message.updated"
	EventMessageDeleted = "message.deleted"
)

// Room helpers centralise naming so producers/consumers agree.
func RoomWorkspace(id string) string { return "ws:" + id }
func RoomList(id string) string      { return "list:" + id }
func RoomTask(id string) string      { return "task:" + id }
func RoomUser(id string) string      { return "user:" + id }
func RoomDoc(id string) string       { return "doc:" + id }
func RoomChannel(id string) string   { return "channel:" + id }
