package doc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
	"github.com/yourorg/clickup/internal/ws"
)

type Service struct {
	repo  domain.DocRepo
	hub   *ws.Hub
	audit *audit.Recorder
}

func New(repo domain.DocRepo, hub *ws.Hub, rec *audit.Recorder) *Service {
	return &Service{repo: repo, hub: hub, audit: rec}
}

type CreateInput struct {
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	SpaceID     *uuid.UUID      `json:"space_id,omitempty"`
	ParentID    *uuid.UUID      `json:"parent_id,omitempty"`
	Title       string          `json:"title,omitempty"`
	Icon        *string         `json:"icon,omitempty"`
	Content     json.RawMessage `json:"content,omitempty"`
	ContentText string          `json:"content_text,omitempty"`
}

type UpdateInput struct {
	Title       *string         `json:"title,omitempty"`
	Icon        *string         `json:"icon,omitempty"`
	Content     json.RawMessage `json:"content,omitempty"`
	ContentText *string         `json:"content_text,omitempty"`
	ParentID    *uuid.UUID      `json:"parent_id,omitempty"`
	SpaceID     *uuid.UUID      `json:"space_id,omitempty"`
	Archived    *bool           `json:"archived,omitempty"`
	OrderIndex  *int            `json:"order_index,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Doc, error) {
	title := in.Title
	if strings.TrimSpace(title) == "" {
		title = "Untitled"
	}
	d := &domain.Doc{
		WorkspaceID: in.WorkspaceID,
		SpaceID:     in.SpaceID,
		ParentID:    in.ParentID,
		CreatorID:   &actor,
		Title:       title,
		Icon:        in.Icon,
		Content:     in.Content,
		ContentText: in.ContentText,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	s.publish(d, ws.EventDocCreated, actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &in.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "doc",
			EntityID:    &d.ID,
			Verb:        "created",
			After:       d,
		})
	}
	return d, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Doc, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Doc, error) {
	return s.repo.ListByWorkspace(ctx, wsID)
}

func (s *Service) ListChildren(ctx context.Context, parentID uuid.UUID) ([]domain.Doc, error) {
	return s.repo.ListChildren(ctx, parentID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Doc, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errors.New("not found")
	}
	if in.Title != nil {
		d.Title = *in.Title
	}
	if in.Icon != nil {
		d.Icon = in.Icon
	}
	if len(in.Content) > 0 {
		d.Content = in.Content
	}
	if in.ContentText != nil {
		d.ContentText = *in.ContentText
	}
	if in.ParentID != nil {
		d.ParentID = in.ParentID
	}
	if in.SpaceID != nil {
		d.SpaceID = in.SpaceID
	}
	if in.Archived != nil {
		d.Archived = *in.Archived
	}
	if in.OrderIndex != nil {
		d.OrderIndex = *in.OrderIndex
	}
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	s.publish(d, ws.EventDocUpdated, actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &d.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "doc",
			EntityID:    &d.ID,
			Verb:        "updated",
		})
	}
	return d, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	d, _ := s.repo.GetByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if d != nil {
		s.publish(d, ws.EventDocDeleted, actor)
		if s.audit != nil {
			s.audit.Record(ctx, audit.Entry{
				WorkspaceID: &d.WorkspaceID,
				ActorID:     &actor,
				EntityType:  "doc",
				EntityID:    &d.ID,
				Verb:        "deleted",
			})
		}
	}
	return nil
}

func (s *Service) publish(d *domain.Doc, kind string, actor uuid.UUID) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(d)
	s.hub.Publish(ws.Event{
		V:           1,
		Room:        ws.RoomDoc(d.ID.String()),
		Type:        kind,
		WorkspaceID: d.WorkspaceID.String(),
		ActorID:     actor.String(),
		EntityID:    d.ID.String(),
		TS:          time.Now().UTC(),
		Payload:     payload,
	})
	// Also broadcast to the workspace room so the sidebar tree refreshes.
	s.hub.Publish(ws.Event{
		V:           1,
		Room:        ws.RoomWorkspace(d.WorkspaceID.String()),
		Type:        kind,
		WorkspaceID: d.WorkspaceID.String(),
		ActorID:     actor.String(),
		EntityID:    d.ID.String(),
		TS:          time.Now().UTC(),
		Payload:     payload,
	})
}
