package customfield

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
	fields domain.CustomFieldRepo
	values domain.CustomValueRepo
	audit  *audit.Recorder
}

func New(fields domain.CustomFieldRepo, values domain.CustomValueRepo, rec *audit.Recorder) *Service {
	return &Service{fields: fields, values: values, audit: rec}
}

type FieldCreateInput struct {
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	ListID      *uuid.UUID      `json:"list_id,omitempty"`
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Config      json.RawMessage `json:"config,omitempty"`
	Required    bool            `json:"required,omitempty"`
	OrderIndex  int             `json:"order_index,omitempty"`
}

type FieldUpdateInput struct {
	Name       *string         `json:"name,omitempty"`
	Kind       *string         `json:"kind,omitempty"`
	Config     json.RawMessage `json:"config,omitempty"`
	Required   *bool           `json:"required,omitempty"`
	OrderIndex *int            `json:"order_index,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in FieldCreateInput) (*domain.CustomField, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	if !domain.IsValidFieldKind(in.Kind) {
		return nil, errors.New("invalid kind")
	}
	config := in.Config
	if len(config) == 0 {
		config = []byte("{}")
	}
	f := &domain.CustomField{
		WorkspaceID: in.WorkspaceID,
		ListID:      in.ListID,
		Name:        strings.TrimSpace(in.Name),
		Kind:        in.Kind,
		Config:      config,
		Required:    in.Required,
		OrderIndex:  in.OrderIndex,
	}
	if err := s.fields.Create(ctx, f); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &in.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "custom_field",
			EntityID:    &f.ID,
			Verb:        "created",
			After:       f,
		})
	}
	return f, nil
}

func (s *Service) ListByList(ctx context.Context, listID uuid.UUID) ([]domain.CustomField, error) {
	return s.fields.ListByList(ctx, listID)
}

func (s *Service) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.CustomField, error) {
	return s.fields.ListByWorkspace(ctx, workspaceID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in FieldUpdateInput) (*domain.CustomField, error) {
	f, err := s.fields.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, errors.New("not found")
	}
	before := *f
	if in.Name != nil {
		f.Name = *in.Name
	}
	if in.Kind != nil {
		if !domain.IsValidFieldKind(*in.Kind) {
			return nil, errors.New("invalid kind")
		}
		f.Kind = *in.Kind
	}
	if len(in.Config) > 0 {
		f.Config = in.Config
	}
	if in.Required != nil {
		f.Required = *in.Required
	}
	if in.OrderIndex != nil {
		f.OrderIndex = *in.OrderIndex
	}
	if err := s.fields.Update(ctx, f); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &f.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "custom_field",
			EntityID:    &f.ID,
			Verb:        "updated",
			Before:      before,
			After:       f,
		})
	}
	return f, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	f, _ := s.fields.GetByID(ctx, id)
	if err := s.fields.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil && f != nil {
		s.audit.Record(ctx, audit.Entry{
			WorkspaceID: &f.WorkspaceID,
			ActorID:     &actor,
			EntityType:  "custom_field",
			EntityID:    &f.ID,
			Verb:        "deleted",
			Before:      f,
		})
	}
	return nil
}

// --- values ------------------------------------------------------------------

type ValueInput struct {
	FieldID uuid.UUID       `json:"field_id"`
	Value   json.RawMessage `json:"value"`
}

func (s *Service) UpsertValue(ctx context.Context, actor, taskID uuid.UUID, in ValueInput) error {
	if in.FieldID == uuid.Nil {
		return errors.New("field_id required")
	}
	v := &domain.CustomValue{
		TaskID:  taskID,
		FieldID: in.FieldID,
		Value:   in.Value,
	}
	if len(v.Value) == 0 {
		v.Value = []byte("null")
	}
	if err := s.values.Upsert(ctx, v); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "custom_value",
			EntityID:   &taskID,
			Verb:       "set",
			After:      v,
		})
	}
	return nil
}

func (s *Service) DeleteValue(ctx context.Context, actor, taskID, fieldID uuid.UUID) error {
	if err := s.values.Delete(ctx, taskID, fieldID); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, audit.Entry{
			ActorID:    &actor,
			EntityType: "custom_value",
			EntityID:   &taskID,
			Verb:       "cleared",
			Before:     map[string]uuid.UUID{"field_id": fieldID},
		})
	}
	return nil
}

func (s *Service) ListValues(ctx context.Context, taskID uuid.UUID) ([]domain.CustomValue, error) {
	return s.values.ListByTask(ctx, taskID)
}
