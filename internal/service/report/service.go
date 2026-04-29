package report

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/domain"
)

type GroupBy string

const (
	GroupByDay  GroupBy = "day"
	GroupByWeek GroupBy = "week"
)

var (
	ErrForbidden = errors.New("forbidden")
)

// WorkspaceAuthorizer abstracts workspace membership so this package doesn't
// depend on the concrete workspace service.
type WorkspaceAuthorizer interface {
	EnsureMember(ctx context.Context, workspaceID, userID uuid.UUID) error
}

// Lookups provides minimal metadata for the export header. Closures keep the
// service decoupled from concrete repositories.
type Lookups struct {
	GetWorkspace func(ctx context.Context, id uuid.UUID) (*domain.Workspace, error)
	GetUser      func(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type Service struct {
	repo    domain.ReportRepo
	ws      WorkspaceAuthorizer
	lookups Lookups
}

func New(repo domain.ReportRepo, ws WorkspaceAuthorizer, lookups Lookups) *Service {
	return &Service{repo: repo, ws: ws, lookups: lookups}
}

type AccomplishmentsInput struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	From        time.Time
	To          time.Time
	GroupBy     GroupBy
}

func (s *Service) Accomplishments(ctx context.Context, in AccomplishmentsInput) ([]domain.AccomplishmentBucket, error) {
	if err := s.ws.EnsureMember(ctx, in.WorkspaceID, in.UserID); err != nil {
		return nil, ErrForbidden
	}
	tasks, err := s.repo.AccomplishedTasks(ctx, domain.AccomplishmentFilter{
		WorkspaceID: in.WorkspaceID,
		UserID:      in.UserID,
		From:        in.From,
		To:          in.To,
	})
	if err != nil {
		return nil, err
	}
	if in.GroupBy == GroupByWeek {
		return bucketByWeek(tasks), nil
	}
	return bucketByDay(tasks), nil
}

func bucketByDay(tasks []domain.AccomplishmentTask) []domain.AccomplishmentBucket {
	groups := map[string][]domain.AccomplishmentTask{}
	starts := map[string]time.Time{}
	for _, t := range tasks {
		ct := t.CompletedAt.UTC()
		key := ct.Format("2006-01-02")
		groups[key] = append(groups[key], t)
		if _, ok := starts[key]; !ok {
			starts[key] = time.Date(ct.Year(), ct.Month(), ct.Day(), 0, 0, 0, 0, time.UTC)
		}
	}
	out := make([]domain.AccomplishmentBucket, 0, len(groups))
	for key, items := range groups {
		start := starts[key]
		out = append(out, domain.AccomplishmentBucket{
			Period:   key,
			StartsAt: start,
			EndsAt:   start.Add(24 * time.Hour),
			Count:    len(items),
			Tasks:    items,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.After(out[j].StartsAt) })
	return out
}

func bucketByWeek(tasks []domain.AccomplishmentTask) []domain.AccomplishmentBucket {
	groups := map[string][]domain.AccomplishmentTask{}
	starts := map[string]time.Time{}
	for _, t := range tasks {
		ct := t.CompletedAt.UTC()
		year, week := ct.ISOWeek()
		key := fmt.Sprintf("%d-W%02d", year, week)
		groups[key] = append(groups[key], t)
		if _, ok := starts[key]; !ok {
			starts[key] = isoWeekStart(year, week)
		}
	}
	out := make([]domain.AccomplishmentBucket, 0, len(groups))
	for key, items := range groups {
		start := starts[key]
		out = append(out, domain.AccomplishmentBucket{
			Period:   key,
			StartsAt: start,
			EndsAt:   start.Add(7 * 24 * time.Hour),
			Count:    len(items),
			Tasks:    items,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.After(out[j].StartsAt) })
	return out
}

// isoWeekStart returns midnight UTC on the Monday of the given ISO week.
func isoWeekStart(year, week int) time.Time {
	// Jan 4th is always in ISO week 1. Walk back to that Monday, then forward
	// (week-1) full weeks. This avoids the year-boundary edge cases that come
	// from naive Jan-1 arithmetic.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	week1Mon := jan4.AddDate(0, 0, 1-weekday)
	return week1Mon.AddDate(0, 0, (week-1)*7)
}
