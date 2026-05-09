package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Portfolio is a workspace-scoped collection of spaces. It exists to
// power the executive rollup view ("everything Mobile is shipping this
// quarter") without forcing each project into one giant space.
type Portfolio struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Color       *string    `json:"color,omitempty"`
	OwnerID     *uuid.UUID `json:"owner_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// SpaceIDs is populated on Get/List so the UI can show "5 spaces"
	// without an extra round-trip. nil-vs-empty matters: nil means "not
	// loaded", []uuid.UUID{} means "loaded, no spaces".
	SpaceIDs []uuid.UUID `json:"space_ids,omitempty"`
}

// PortfolioRollup is the per-portfolio aggregate for the rollup view.
// All counts are workspace-scoped via the spaces in the portfolio.
type PortfolioRollup struct {
	PortfolioID    uuid.UUID `json:"portfolio_id"`
	SpaceCount     int       `json:"space_count"`
	ListCount      int       `json:"list_count"`
	TotalTasks     int       `json:"total_tasks"`
	OpenTasks      int       `json:"open_tasks"`
	CompletedTasks int       `json:"completed_tasks"`
	OverdueTasks   int       `json:"overdue_tasks"`
	// DueThisWeek is open tasks whose due_at falls in (now, now+7d).
	// Useful for "what's shipping this week" exec digests.
	DueThisWeek int `json:"due_this_week"`
}

type PortfolioRepo interface {
	Create(ctx context.Context, p *Portfolio) error
	Get(ctx context.Context, id uuid.UUID) (*Portfolio, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Portfolio, error)
	Update(ctx context.Context, p *Portfolio) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Space linkage. AttachSpace is idempotent.
	AttachSpace(ctx context.Context, portfolioID, spaceID uuid.UUID) error
	DetachSpace(ctx context.Context, portfolioID, spaceID uuid.UUID) error
	ListSpaces(ctx context.Context, portfolioID uuid.UUID) ([]uuid.UUID, error)

	// Rollup is the aggregated counts query. Single SQL pass.
	Rollup(ctx context.Context, portfolioID uuid.UUID) (*PortfolioRollup, error)
}
