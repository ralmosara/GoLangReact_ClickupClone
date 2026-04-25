package view

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
)

type Service struct {
	repo  domain.ViewRepo
	audit *audit.Recorder
}

func New(repo domain.ViewRepo, rec *audit.Recorder) *Service {
	return &Service{repo: repo, audit: rec}
}

type CreateInput struct {
	ListID  *uuid.UUID         `json:"list_id,omitempty"`
	SpaceID *uuid.UUID         `json:"space_id,omitempty"`
	Name    string             `json:"name"`
	Kind    string             `json:"kind"`
	Config  *domain.ViewConfig `json:"config,omitempty"`
}

type UpdateInput struct {
	Name   *string            `json:"name,omitempty"`
	Kind   *string            `json:"kind,omitempty"`
	Config *domain.ViewConfig `json:"config,omitempty"`
}

var validKinds = map[string]bool{
	domain.ViewKindBoard:    true,
	domain.ViewKindList:     true,
	domain.ViewKindCalendar: true,
	domain.ViewKindGantt:    true,
	domain.ViewKindTable:    true,
	domain.ViewKindTimeline: true,
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.View, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	if in.ListID == nil && in.SpaceID == nil {
		return nil, errors.New("list_id or space_id required")
	}
	kind := in.Kind
	if kind == "" {
		kind = domain.ViewKindBoard
	}
	if !validKinds[kind] {
		return nil, errors.New("unknown view kind")
	}
	cfg := domain.ViewConfig{}
	if in.Config != nil {
		cfg = *in.Config
	}
	cfg.CreatorID = &actor
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	v := &domain.View{
		ListID:  in.ListID,
		SpaceID: in.SpaceID,
		Name:    in.Name,
		Kind:    kind,
		Config:  configJSON,
	}
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "view",
			EntityID:   &v.ID,
			Verb:       "created",
			After:      v,
		})
	}
	return v, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.View, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.View, error) {
	return s.repo.ListByList(ctx, listID)
}

func (s *Service) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]domain.View, error) {
	return s.repo.ListBySpace(ctx, spaceID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.View, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, errors.New("not found")
	}
	before := *v
	if in.Name != nil {
		v.Name = *in.Name
	}
	if in.Kind != nil {
		if !validKinds[*in.Kind] {
			return nil, errors.New("unknown view kind")
		}
		v.Kind = *in.Kind
	}
	if in.Config != nil {
		if in.Config.CreatorID == nil {
			existing, _ := v.DecodeConfig()
			in.Config.CreatorID = existing.CreatorID
		}
		b, err := json.Marshal(*in.Config)
		if err != nil {
			return nil, err
		}
		v.Config = b
	}
	if err := s.repo.Update(ctx, v); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "view",
			EntityID:   &v.ID,
			Verb:       "updated",
			Before:     before,
			After:      v,
		})
	}
	return v, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	v, _ := s.repo.GetByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil && v != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "view",
			EntityID:   &v.ID,
			Verb:       "deleted",
			Before:     v,
		})
	}
	return nil
}
