package whiteboard

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
	repo  domain.WhiteboardRepo
	hub   *ws.Hub
	audit *audit.Recorder
}

func New(repo domain.WhiteboardRepo, hub *ws.Hub, rec *audit.Recorder) *Service {
	return &Service{repo: repo, hub: hub, audit: rec}
}

type CreateInput struct {
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	SpaceID     *uuid.UUID      `json:"space_id,omitempty"`
	Name        string          `json:"name"`
	Snapshot    json.RawMessage `json:"snapshot,omitempty"`
}

type UpdateInput struct {
	Name     *string         `json:"name,omitempty"`
	Snapshot json.RawMessage `json:"snapshot,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Whiteboard, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	w := &domain.Whiteboard{
		WorkspaceID: in.WorkspaceID,
		SpaceID:     in.SpaceID,
		Name:        strings.TrimSpace(in.Name),
		Snapshot:    in.Snapshot,
		CreatorID:   &actor,
	}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, err
	}
	s.publish(w, "whiteboard.created", actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &w.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "whiteboard",
			EntityID:    &w.ID,
			Verb:        "created",
			After:       w,
		})
	}
	return w, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Whiteboard, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Whiteboard, error) {
	return s.repo.ListByWorkspace(ctx, wsID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Whiteboard, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, errors.New("not found")
	}
	if in.Name != nil {
		w.Name = *in.Name
	}
	if len(in.Snapshot) > 0 {
		w.Snapshot = in.Snapshot
	}
	if err := s.repo.Update(ctx, w); err != nil {
		return nil, err
	}
	s.publish(w, "whiteboard.updated", actor)
	return w, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	w, _ := s.repo.GetByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if w != nil {
		s.publish(w, "whiteboard.deleted", actor)
	}
	return nil
}

func (s *Service) publish(w *domain.Whiteboard, kind string, actor uuid.UUID) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"id":      w.ID,
		"name":    w.Name,
		"version": w.Version,
	})
	s.hub.Publish(ws.Event{
		V:           1,
		Room:        "whiteboard:" + w.ID.String(),
		Type:        kind,
		WorkspaceID: w.WorkspaceID.String(),
		ActorID:     actor.String(),
		EntityID:    w.ID.String(),
		TS:          time.Now().UTC(),
		Payload:     payload,
	})
	// Also broadcast to the workspace room so the sidebar list refreshes.
	s.hub.Publish(ws.Event{
		V:           1,
		Room:        ws.RoomWorkspace(w.WorkspaceID.String()),
		Type:        kind,
		WorkspaceID: w.WorkspaceID.String(),
		ActorID:     actor.String(),
		EntityID:    w.ID.String(),
		TS:          time.Now().UTC(),
		Payload:     payload,
	})
}
