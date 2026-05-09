// Package savedsearch is the per-user saved-search bookmark service.
// CRUD only — running a saved search just feeds its (q, entity_type,
// filters) back into the existing search service.
//
// Authorisation: every operation is keyed by user_id; cross-user reads
// are not expressible via the public methods. The handler also confirms
// the URL workspace matches the saved row's workspace_id before
// surfacing it.
package savedsearch

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

type Service struct{ repo domain.SavedSearchRepo }

func New(repo domain.SavedSearchRepo) *Service { return &Service{repo: repo} }

type CreateInput struct {
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	Name        string          `json:"name"`
	EntityType  *string         `json:"entity_type,omitempty"`
	QueryText   string          `json:"query_text"`
	Filters     json.RawMessage `json:"filters"`
	Pinned      bool            `json:"pinned"`
}

type UpdateInput struct {
	Name       string          `json:"name"`
	EntityType *string         `json:"entity_type,omitempty"`
	QueryText  string          `json:"query_text"`
	Filters    json.RawMessage `json:"filters"`
	Pinned     bool            `json:"pinned"`
}

var (
	ErrNameRequired = errors.New("name is required")
	ErrNotOwner     = errors.New("saved search does not belong to caller")
	ErrNotFound     = errors.New("saved search not found")
)

func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput) (*domain.SavedSearch, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, ErrNameRequired
	}
	row := &domain.SavedSearch{
		UserID:      userID,
		WorkspaceID: in.WorkspaceID,
		Name:        strings.TrimSpace(in.Name),
		EntityType:  in.EntityType,
		QueryText:   in.QueryText,
		Filters:     in.Filters,
		Pinned:      in.Pinned,
	}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) List(ctx context.Context, userID, workspaceID uuid.UUID) ([]domain.SavedSearch, error) {
	return s.repo.ListForUser(ctx, userID, workspaceID)
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput) (*domain.SavedSearch, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, ErrNameRequired
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrNotFound
	}
	if existing.UserID != userID {
		return nil, ErrNotOwner
	}
	existing.Name = strings.TrimSpace(in.Name)
	existing.EntityType = in.EntityType
	existing.QueryText = in.QueryText
	existing.Filters = in.Filters
	existing.Pinned = in.Pinned
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil // idempotent
	}
	if existing.UserID != userID {
		return ErrNotOwner
	}
	return s.repo.Delete(ctx, id)
}

// MarkUsed bumps last_used_at — the UI calls it after running a saved
// search so the recent bucket reflects actual use.
func (s *Service) MarkUsed(ctx context.Context, userID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil || existing.UserID != userID {
		return nil
	}
	return s.repo.TouchUsed(ctx, id)
}
