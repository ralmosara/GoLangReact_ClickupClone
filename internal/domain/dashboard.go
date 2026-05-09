package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Widget kinds — the frontend picks a component per kind. New kinds:
//
//   * my_tasks    — tasks assigned to the calling user, grouped by due
//                   bucket (overdue / today / this_week / later). Config
//                   is just { workspace_id }; viewer = caller.
//   * activity    — recent activity rows pulled off audit_log, scoped
//                   to the workspace. Config { workspace_id, limit? }.
//   * goal_progress — per-goal % complete + due-at, sorted by due-soon.
//                     Config { workspace_id }.
const (
	WidgetKindBurndown     = "burndown"
	WidgetKindVelocity     = "velocity"
	WidgetKindTaskCount    = "task_count"
	WidgetKindTimePerUser  = "time_per_user"
	WidgetKindMyTasks      = "my_tasks"
	WidgetKindActivity     = "activity"
	WidgetKindGoalProgress = "goal_progress"
)

type Dashboard struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	SpaceID     *uuid.UUID `json:"space_id,omitempty"`
	Name        string     `json:"name"`
	CreatorID   *uuid.UUID `json:"creator_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Widget struct {
	ID          uuid.UUID       `json:"id"`
	DashboardID uuid.UUID       `json:"dashboard_id"`
	Kind        string          `json:"kind"`
	Title       string          `json:"title"`
	Config      json.RawMessage `json:"config"`
	OrderIndex  int             `json:"order_index"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type DashboardRepo interface {
	Create(ctx context.Context, d *Dashboard) error
	GetByID(ctx context.Context, id uuid.UUID) (*Dashboard, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Dashboard, error)
	Update(ctx context.Context, d *Dashboard) error
	Delete(ctx context.Context, id uuid.UUID) error

	CreateWidget(ctx context.Context, w *Widget) error
	ListWidgets(ctx context.Context, dashboardID uuid.UUID) ([]Widget, error)
	UpdateWidget(ctx context.Context, w *Widget) error
	DeleteWidget(ctx context.Context, id uuid.UUID) error
	GetWidget(ctx context.Context, id uuid.UUID) (*Widget, error)
}
