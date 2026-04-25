package status

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
	repo  domain.StatusRepo
	hub   *ws.Hub
	audit *audit.Recorder
}

func New(repo domain.StatusRepo, hub *ws.Hub, rec *audit.Recorder) *Service {
	return &Service{repo: repo, hub: hub, audit: rec}
}

type CreateInput struct {
	ListID     uuid.UUID `json:"list_id"`
	Name       string    `json:"name"`
	Color      string    `json:"color"`
	Category   string    `json:"category"`
	OrderIndex int       `json:"order_index"`
}

type UpdateInput struct {
	Name       *string `json:"name,omitempty"`
	Color      *string `json:"color,omitempty"`
	Category   *string `json:"category,omitempty"`
	OrderIndex *int    `json:"order_index,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Status, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	st := &domain.Status{
		ListID:     in.ListID,
		Name:       in.Name,
		Color:      nonEmpty(in.Color, "#94a3b8"),
		Category:   nonEmpty(in.Category, "active"),
		OrderIndex: in.OrderIndex,
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, err
	}
	s.publish(st, ws.EventStatusCreated, actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "status",
			EntityID:   &st.ID,
			Verb:       "created",
			After:      st,
		})
	}
	return st, nil
}

func (s *Service) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.Status, error) {
	return s.repo.ListByList(ctx, listID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Status, error) {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, errors.New("not found")
	}
	before := *st
	if in.Name != nil {
		st.Name = *in.Name
	}
	if in.Color != nil {
		st.Color = *in.Color
	}
	if in.Category != nil {
		st.Category = *in.Category
	}
	if in.OrderIndex != nil {
		st.OrderIndex = *in.OrderIndex
	}
	if err := s.repo.Update(ctx, st); err != nil {
		return nil, err
	}
	s.publish(st, ws.EventStatusUpdated, actor)
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "status",
			EntityID:   &st.ID,
			Verb:       "updated",
			Before:     before,
			After:      st,
		})
	}
	return st, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if st != nil {
		s.publish(st, ws.EventStatusDeleted, actor)
		if s.audit != nil {
			s.audit.Record(ctx, audit.Entry{
				ActorID:    &actor,
				EntityType: "status",
				EntityID:   &st.ID,
				Verb:       "deleted",
				Before:     st,
			})
		}
	}
	return nil
}

func (s *Service) publish(st *domain.Status, kind string, actor uuid.UUID) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(st)
	s.hub.Publish(ws.Event{
		V:        1,
		Room:     ws.RoomList(st.ListID.String()),
		Type:     kind,
		ActorID:  actor.String(),
		EntityID: st.ID.String(),
		TS:       time.Now().UTC(),
		Payload:  payload,
	})
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
