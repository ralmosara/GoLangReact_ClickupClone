package workspace

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

type Service struct {
	repo domain.WorkspaceRepo
}

func New(repo domain.WorkspaceRepo) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, in CreateInput) (*domain.Workspace, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("name required")
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		slug = strings.ToLower(name)
	}
	slug = slugRe.ReplaceAllString(strings.ReplaceAll(slug, " ", "-"), "")
	if slug == "" {
		slug = ownerID.String()[:8]
	}

	w := &domain.Workspace{Name: name, Slug: slug, OwnerID: ownerID}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, err
	}
	if err := s.repo.AddMember(ctx, &domain.WorkspaceMember{
		WorkspaceID: w.ID, UserID: ownerID, Role: "owner",
	}); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	return s.repo.ListForUser(ctx, userID)
}

func (s *Service) EnsureMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	ok, err := s.repo.IsMember(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not a member")
	}
	return nil
}
