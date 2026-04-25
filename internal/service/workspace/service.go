package workspace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yourorg/clickup/internal/domain"
)

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// Postgres SQLSTATE for unique_violation.
const pgUniqueViolation = "23505"

const slugCreateRetries = 5

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
	userProvidedSlug := strings.TrimSpace(in.Slug) != ""
	base := normalizeSlug(in.Slug)
	if base == "" {
		base = normalizeSlug(name)
	}
	if base == "" {
		base = ownerID.String()[:8]
	}

	// Slug is globally unique. If the user picked one explicitly, surface the
	// collision; if we derived it from the name, retry with a short suffix so
	// "Marketing" → "marketing-a3f2" instead of a 400.
	w := &domain.Workspace{Name: name, OwnerID: ownerID}
	for attempt := 0; attempt < slugCreateRetries; attempt++ {
		w.Slug = base
		if attempt > 0 {
			w.Slug = base + "-" + randomSuffix(4)
		}
		err := s.repo.Create(ctx, w)
		if err == nil {
			break
		}
		if !isUniqueViolation(err) || userProvidedSlug || attempt == slugCreateRetries-1 {
			return nil, err
		}
	}

	if err := s.repo.AddMember(ctx, &domain.WorkspaceMember{
		WorkspaceID: w.ID, UserID: ownerID, Role: "owner",
	}); err != nil {
		return nil, err
	}
	return w, nil
}

func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return slugRe.ReplaceAllString(s, "")
}

func randomSuffix(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failures are effectively impossible on supported OSes;
		// fall back to a uuid fragment so we still produce *something* unique.
		return uuid.New().String()[:nBytes*2]
	}
	return hex.EncodeToString(b)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
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
