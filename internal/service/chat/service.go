package chat

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/notify"
	"github.com/yourorg/clickup/internal/ws"
)

type Service struct {
	channels domain.ChannelRepo
	messages domain.MessageRepo
	hub      *ws.Hub
	notify   *notify.Dispatcher
	audit    *audit.Recorder
}

type Deps struct {
	Channels domain.ChannelRepo
	Messages domain.MessageRepo
	Hub      *ws.Hub
	Notify   *notify.Dispatcher
	Audit    *audit.Recorder
}

func New(d Deps) *Service {
	return &Service{
		channels: d.Channels,
		messages: d.Messages,
		hub:      d.Hub,
		notify:   d.Notify,
		audit:    d.Audit,
	}
}

/* ------ channels ---------------------------------------------------------- */

type CreateChannelInput struct {
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	SpaceID     *uuid.UUID `json:"space_id,omitempty"`
	Name        string     `json:"name"`
	Topic       string     `json:"topic,omitempty"`
	Kind        string     `json:"kind,omitempty"`
	IsPrivate   bool       `json:"is_private,omitempty"`
}

type UpdateChannelInput struct {
	Name      *string `json:"name,omitempty"`
	Topic     *string `json:"topic,omitempty"`
	IsPrivate *bool   `json:"is_private,omitempty"`
}

func (s *Service) CreateChannel(ctx context.Context, actor uuid.UUID, in CreateChannelInput) (*domain.Channel, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	c := &domain.Channel{
		WorkspaceID: in.WorkspaceID,
		SpaceID:     in.SpaceID,
		Name:        strings.TrimSpace(in.Name),
		Topic:       in.Topic,
		Kind:        in.Kind,
		IsPrivate:   in.IsPrivate,
		CreatorID:   &actor,
	}
	if err := s.channels.Create(ctx, c); err != nil {
		return nil, err
	}
	_ = s.channels.AddMember(ctx, c.ID, actor) // creator is always a member
	s.publishChannel(c, ws.EventChannelCreated, actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &c.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "channel",
			EntityID:    &c.ID,
			Verb:        "created",
			After:       c,
		})
	}
	return c, nil
}

func (s *Service) GetChannel(ctx context.Context, id uuid.UUID) (*domain.Channel, error) {
	return s.channels.GetByID(ctx, id)
}

func (s *Service) ListChannelsByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Channel, error) {
	return s.channels.ListByWorkspace(ctx, wsID)
}

func (s *Service) ListChannelsBySpace(ctx context.Context, spaceID uuid.UUID) ([]domain.Channel, error) {
	return s.channels.ListBySpace(ctx, spaceID)
}

func (s *Service) UpdateChannel(ctx context.Context, actor, id uuid.UUID, in UpdateChannelInput) (*domain.Channel, error) {
	c, err := s.channels.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errors.New("not found")
	}
	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.Topic != nil {
		c.Topic = *in.Topic
	}
	if in.IsPrivate != nil {
		c.IsPrivate = *in.IsPrivate
	}
	if err := s.channels.Update(ctx, c); err != nil {
		return nil, err
	}
	s.publishChannel(c, ws.EventChannelUpdated, actor)
	return c, nil
}

func (s *Service) DeleteChannel(ctx context.Context, actor, id uuid.UUID) error {
	c, _ := s.channels.GetByID(ctx, id)
	if err := s.channels.Delete(ctx, id); err != nil {
		return err
	}
	if c != nil {
		s.publishChannel(c, ws.EventChannelDeleted, actor)
	}
	return nil
}

func (s *Service) AddMember(ctx context.Context, channelID, userID uuid.UUID) error {
	return s.channels.AddMember(ctx, channelID, userID)
}

func (s *Service) RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error {
	return s.channels.RemoveMember(ctx, channelID, userID)
}

func (s *Service) ListMembers(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	return s.channels.ListMembers(ctx, channelID)
}

func (s *Service) MarkRead(ctx context.Context, channelID, userID uuid.UUID) error {
	return s.channels.TouchRead(ctx, channelID, userID)
}

/* ------ messages ---------------------------------------------------------- */

type PostMessageInput struct {
	Body            string      `json:"body"`
	ParentMessageID *uuid.UUID  `json:"parent_message_id,omitempty"`
	MentionIDs      []uuid.UUID `json:"mention_ids,omitempty"`
}

var mentionRE = regexp.MustCompile(`@([a-zA-Z0-9._+-]+@[a-zA-Z0-9.-]+)`)

func (s *Service) PostMessage(ctx context.Context, actor, channelID uuid.UUID, in PostMessageInput) (*domain.Message, error) {
	if strings.TrimSpace(in.Body) == "" {
		return nil, errors.New("body required")
	}
	m := &domain.Message{
		ChannelID:       channelID,
		AuthorID:        &actor,
		ParentMessageID: in.ParentMessageID,
		Body:            in.Body,
	}
	if err := s.messages.Create(ctx, m); err != nil {
		return nil, err
	}
	s.publishMessage(m, ws.EventMessageCreated, actor)

	// Fan out @mentions to the notify dispatcher.
	if s.notify != nil {
		ch, _ := s.channels.GetByID(ctx, channelID)
		seen := map[uuid.UUID]bool{actor: true}
		for _, uid := range in.MentionIDs {
			if seen[uid] {
				continue
			}
			seen[uid] = true
			payload := map[string]any{
				"channel_id": channelID,
				"message_id": m.ID,
				"excerpt":    snippet(in.Body, 140),
			}
			if ch != nil {
				payload["channel_name"] = ch.Name
			}
			mid := m.ID
			s.notify.Enqueue(notify.Notice{
				Recipient:  uid,
				Actor:      &actor,
				Kind:       "message.mention",
				EntityType: "message",
				EntityID:   &mid,
				Payload:    payload,
			})
		}
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "message",
			EntityID:   &m.ID,
			Verb:       "posted",
		})
	}
	return m, nil
}

func (s *Service) ListMessages(ctx context.Context, channelID uuid.UUID, parentID *uuid.UUID, limit int, before *time.Time) ([]domain.Message, error) {
	return s.messages.List(ctx, domain.MessageFilter{
		ChannelID: channelID,
		ParentID:  parentID,
		Before:    before,
		Limit:     limit,
	})
}

// ListReplies walks a thread. The parent's channel is resolved first so the
// handler only needs the parent message id.
func (s *Service) ListReplies(ctx context.Context, parentID uuid.UUID, limit int) ([]domain.Message, error) {
	parent, err := s.messages.GetByID(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, errors.New("parent not found")
	}
	return s.messages.List(ctx, domain.MessageFilter{
		ChannelID: parent.ChannelID,
		ParentID:  &parentID,
		Limit:     limit,
	})
}

type EditMessageInput struct {
	Body string `json:"body"`
}

func (s *Service) EditMessage(ctx context.Context, actor, id uuid.UUID, in EditMessageInput) (*domain.Message, error) {
	m, err := s.messages.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("not found")
	}
	if m.AuthorID == nil || *m.AuthorID != actor {
		return nil, errors.New("forbidden")
	}
	m.Body = in.Body
	if err := s.messages.Update(ctx, m); err != nil {
		return nil, err
	}
	s.publishMessage(m, ws.EventMessageUpdated, actor)
	return m, nil
}

func (s *Service) DeleteMessage(ctx context.Context, actor, id uuid.UUID) error {
	m, err := s.messages.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return nil
	}
	if m.AuthorID == nil || *m.AuthorID != actor {
		return errors.New("forbidden")
	}
	if err := s.messages.SoftDelete(ctx, id); err != nil {
		return err
	}
	m.DeletedAt = ptr(time.Now().UTC())
	m.Body = ""
	s.publishMessage(m, ws.EventMessageDeleted, actor)
	return nil
}

/* ------ ws publish helpers ----------------------------------------------- */

func (s *Service) publishChannel(c *domain.Channel, kind string, actor uuid.UUID) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(c)
	s.hub.Publish(ws.Event{
		V:           1,
		Room:        ws.RoomWorkspace(c.WorkspaceID.String()),
		Type:        kind,
		WorkspaceID: c.WorkspaceID.String(),
		ActorID:     actor.String(),
		EntityID:    c.ID.String(),
		TS:          time.Now().UTC(),
		Payload:     payload,
	})
}

func (s *Service) publishMessage(m *domain.Message, kind string, actor uuid.UUID) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(m)
	s.hub.Publish(ws.Event{
		V:        1,
		Room:     ws.RoomChannel(m.ChannelID.String()),
		Type:     kind,
		ActorID:  actor.String(),
		EntityID: m.ID.String(),
		TS:       time.Now().UTC(),
		Payload:  payload,
	})
}

func snippet(body string, max int) string {
	body = strings.TrimSpace(body)
	if len([]rune(body)) <= max {
		return body
	}
	runes := []rune(body)
	return string(runes[:max]) + "…"
}

func ptr[T any](v T) *T { return &v }

// ExtractMentionHandles returns the @handles found in a message body. Currently
// only parses `@email@host` style; the frontend handles the UUID lookup and
// sends explicit IDs via PostMessageInput.MentionIDs.
func ExtractMentionHandles(body string) []string {
	out := mentionRE.FindAllStringSubmatch(body, -1)
	handles := make([]string, 0, len(out))
	for _, m := range out {
		handles = append(handles, m[1])
	}
	return handles
}
