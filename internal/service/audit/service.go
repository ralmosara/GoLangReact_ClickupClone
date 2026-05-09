package audit

import (
	"context"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

// FacetsProvider is the optional interface the concrete repo implements to
// surface the distinct entity_types/verbs/actor_ids in a workspace. We
// keep it optional rather than promoting it to domain.AuditRepo so a mock
// or alternative impl doesn't have to ship the SQL ARRAY_AGG.
type FacetsProvider interface {
	ListFacets(ctx context.Context, workspaceID uuid.UUID) (*Facets, error)
}

// Facets carries the distinct values for the audit filters in a workspace.
// JSON shape is shared with the handler so the UI can decode directly.
type Facets struct {
	EntityTypes []string    `json:"entity_types"`
	Verbs       []string    `json:"verbs"`
	ActorIDs    []uuid.UUID `json:"actor_ids"`
}

type Service struct {
	repo domain.AuditRepo
}

func New(repo domain.AuditRepo) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f domain.AuditFilter) ([]domain.AuditEntry, error) {
	return s.repo.List(ctx, f)
}

// Facets is a soft-typed pass-through to the repo if it implements
// FacetsProvider. Returns (nil, nil) — not an error — when the repo does
// not, so the handler can decide whether to 501 or fall back gracefully.
func (s *Service) Facets(ctx context.Context, workspaceID uuid.UUID) (*Facets, error) {
	fp, ok := s.repo.(FacetsProvider)
	if !ok {
		return nil, nil
	}
	return fp.ListFacets(ctx, workspaceID)
}
