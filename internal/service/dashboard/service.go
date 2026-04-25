package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yourorg/clickup/internal/audit"
	"github.com/yourorg/clickup/internal/domain"
)

// Data-providers decouple widgets from concrete services so the dashboard
// package doesn't reach into sprint/time/task services directly (keeping imports
// one-directional). main.go wires these closures.
type DataProviders struct {
	Burndown    func(ctx context.Context, sprintID uuid.UUID) ([]domain.BurndownPoint, error)
	TaskCount   func(ctx context.Context, listID uuid.UUID) (map[string]int, error)
	Velocity    func(ctx context.Context, listID uuid.UUID) ([]VelocityPoint, error)
	TimePerUser func(ctx context.Context, workspaceID uuid.UUID, from, to *time.Time) ([]domain.TimeReportBucket, error)
}

type VelocityPoint struct {
	SprintID   uuid.UUID `json:"sprint_id"`
	SprintName string    `json:"sprint_name"`
	EndsAt     time.Time `json:"ends_at"`
	Completed  int       `json:"completed_points"`
	Goal       int       `json:"goal_points"`
}

type Service struct {
	repo      domain.DashboardRepo
	providers DataProviders
	audit     *audit.Recorder
}

func New(repo domain.DashboardRepo, providers DataProviders, rec *audit.Recorder) *Service {
	return &Service{repo: repo, providers: providers, audit: rec}
}

/* --- dashboards ---------------------------------------------------------- */

type CreateInput struct {
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	SpaceID     *uuid.UUID `json:"space_id,omitempty"`
	Name        string     `json:"name"`
}

type UpdateInput struct {
	Name    *string    `json:"name,omitempty"`
	SpaceID *uuid.UUID `json:"space_id,omitempty"`
}

func (s *Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (*domain.Dashboard, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("name required")
	}
	d := &domain.Dashboard{
		WorkspaceID: in.WorkspaceID,
		SpaceID:     in.SpaceID,
		Name:        strings.TrimSpace(in.Name),
		CreatorID:   &actor,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Dashboard, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Dashboard, error) {
	return s.repo.ListByWorkspace(ctx, wsID)
}

func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (*domain.Dashboard, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errors.New("not found")
	}
	if in.Name != nil {
		d.Name = *in.Name
	}
	if in.SpaceID != nil {
		d.SpaceID = in.SpaceID
	}
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) Delete(ctx context.Context, actor, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

/* --- widgets ------------------------------------------------------------ */

type CreateWidgetInput struct {
	Kind       string          `json:"kind"`
	Title      string          `json:"title"`
	Config     json.RawMessage `json:"config,omitempty"`
	OrderIndex int             `json:"order_index,omitempty"`
}

type UpdateWidgetInput struct {
	Kind       *string         `json:"kind,omitempty"`
	Title      *string         `json:"title,omitempty"`
	Config     json.RawMessage `json:"config,omitempty"`
	OrderIndex *int            `json:"order_index,omitempty"`
}

var validWidgets = map[string]bool{
	domain.WidgetKindBurndown:    true,
	domain.WidgetKindVelocity:    true,
	domain.WidgetKindTaskCount:   true,
	domain.WidgetKindTimePerUser: true,
}

func (s *Service) AddWidget(ctx context.Context, dashboardID uuid.UUID, in CreateWidgetInput) (*domain.Widget, error) {
	if !validWidgets[in.Kind] {
		return nil, errors.New("invalid widget kind")
	}
	w := &domain.Widget{
		DashboardID: dashboardID,
		Kind:        in.Kind,
		Title:       in.Title,
		Config:      in.Config,
		OrderIndex:  in.OrderIndex,
	}
	if err := s.repo.CreateWidget(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) ListWidgets(ctx context.Context, dashboardID uuid.UUID) ([]domain.Widget, error) {
	return s.repo.ListWidgets(ctx, dashboardID)
}

func (s *Service) UpdateWidget(ctx context.Context, id uuid.UUID, in UpdateWidgetInput) (*domain.Widget, error) {
	w, err := s.repo.GetWidget(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, errors.New("not found")
	}
	if in.Kind != nil {
		if !validWidgets[*in.Kind] {
			return nil, errors.New("invalid widget kind")
		}
		w.Kind = *in.Kind
	}
	if in.Title != nil {
		w.Title = *in.Title
	}
	if len(in.Config) > 0 {
		w.Config = in.Config
	}
	if in.OrderIndex != nil {
		w.OrderIndex = *in.OrderIndex
	}
	if err := s.repo.UpdateWidget(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) DeleteWidget(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteWidget(ctx, id)
}

/* --- widget data --------------------------------------------------------- */

// WidgetData resolves a widget's data by dispatching on kind + config.
// Widget config shape:
//   burndown:      { sprint_id: UUID }
//   velocity:      { list_id: UUID, last?: int }
//   task_count:    { list_id: UUID }
//   time_per_user: { workspace_id: UUID, from?: RFC3339, to?: RFC3339 }
func (s *Service) WidgetData(ctx context.Context, widgetID uuid.UUID) (any, error) {
	w, err := s.repo.GetWidget(ctx, widgetID)
	if err != nil || w == nil {
		return nil, err
	}
	var cfg map[string]any
	if len(w.Config) > 0 {
		_ = json.Unmarshal(w.Config, &cfg)
	}
	switch w.Kind {
	case domain.WidgetKindBurndown:
		id, ok := parseUUID(cfg["sprint_id"])
		if !ok {
			return nil, errors.New("sprint_id required")
		}
		return s.providers.Burndown(ctx, id)
	case domain.WidgetKindTaskCount:
		id, ok := parseUUID(cfg["list_id"])
		if !ok {
			return nil, errors.New("list_id required")
		}
		return s.providers.TaskCount(ctx, id)
	case domain.WidgetKindVelocity:
		id, ok := parseUUID(cfg["list_id"])
		if !ok {
			return nil, errors.New("list_id required")
		}
		return s.providers.Velocity(ctx, id)
	case domain.WidgetKindTimePerUser:
		id, ok := parseUUID(cfg["workspace_id"])
		if !ok {
			return nil, errors.New("workspace_id required")
		}
		from := parseTimeP(cfg["from"])
		to := parseTimeP(cfg["to"])
		return s.providers.TimePerUser(ctx, id, from, to)
	}
	return nil, errors.New("unknown widget kind")
}

func parseUUID(v any) (uuid.UUID, bool) {
	s, ok := v.(string)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func parseTimeP(v any) *time.Time {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
