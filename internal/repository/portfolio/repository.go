package portfolio

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const cols = `id, workspace_id, name, description, color, owner_id, created_at, updated_at`

func (r *Repo) Create(ctx context.Context, p *domain.Portfolio) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO portfolios (workspace_id, name, description, color, owner_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, p.WorkspaceID, p.Name, p.Description, p.Color, p.OwnerID).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (*domain.Portfolio, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+cols+` FROM portfolios WHERE id = $1`, id)
	p, err := scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	spaceIDs, err := r.ListSpaces(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.SpaceIDs = spaceIDs
	return p, nil
}

func (r *Repo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Portfolio, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+cols+` FROM portfolios WHERE workspace_id = $1 ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var portfolios []domain.Portfolio
	for rows.Next() {
		p, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		portfolios = append(portfolios, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Hydrate space_ids in one batch query so we don't N+1 the listing.
	if len(portfolios) > 0 {
		ids := make([]uuid.UUID, 0, len(portfolios))
		for _, p := range portfolios {
			ids = append(ids, p.ID)
		}
		spacesByPortfolio, err := r.spacesByPortfolio(ctx, ids)
		if err != nil {
			return nil, err
		}
		for i := range portfolios {
			portfolios[i].SpaceIDs = spacesByPortfolio[portfolios[i].ID]
		}
	}
	return portfolios, nil
}

// spacesByPortfolio returns a map portfolio_id → []space_id for the
// supplied portfolios in a single round-trip.
func (r *Repo) spacesByPortfolio(ctx context.Context, portfolioIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT portfolio_id, space_id FROM portfolio_spaces
		 WHERE portfolio_id = ANY($1::uuid[])
		 ORDER BY portfolio_id, added_at ASC
	`, portfolioIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID][]uuid.UUID, len(portfolioIDs))
	for rows.Next() {
		var pid, sid uuid.UUID
		if err := rows.Scan(&pid, &sid); err != nil {
			return nil, err
		}
		out[pid] = append(out[pid], sid)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, p *domain.Portfolio) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE portfolios SET
		  name = $2, description = $3, color = $4, owner_id = $5, updated_at = NOW()
		WHERE id = $1
	`, p.ID, p.Name, p.Description, p.Color, p.OwnerID)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM portfolios WHERE id = $1`, id)
	return err
}

func (r *Repo) AttachSpace(ctx context.Context, portfolioID, spaceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO portfolio_spaces (portfolio_id, space_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, portfolioID, spaceID)
	return err
}

func (r *Repo) DetachSpace(ctx context.Context, portfolioID, spaceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM portfolio_spaces WHERE portfolio_id = $1 AND space_id = $2`,
		portfolioID, spaceID,
	)
	return err
}

func (r *Repo) ListSpaces(ctx context.Context, portfolioID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT space_id FROM portfolio_spaces
		 WHERE portfolio_id = $1
		 ORDER BY added_at ASC
	`, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Rollup walks the portfolio → spaces → lists → tasks chain in one
// CTE, then GROUP BYs to produce the executive summary line. Bounded
// by portfolio size so the query stays fast even on large workspaces
// (typical portfolio has 5-15 spaces).
func (r *Repo) Rollup(ctx context.Context, portfolioID uuid.UUID) (*domain.PortfolioRollup, error) {
	const q = `
		WITH ps AS (
		  SELECT space_id FROM portfolio_spaces WHERE portfolio_id = $1
		),
		ls AS (
		  SELECT id FROM lists WHERE space_id IN (SELECT space_id FROM ps)
		),
		ts AS (
		  SELECT t.* FROM tasks t
		   WHERE t.list_id IN (SELECT id FROM ls)
		     AND t.archived = FALSE
		     AND t.parent_task_id IS NULL
		)
		SELECT
		  (SELECT COUNT(*) FROM ps)::int                                                     AS space_count,
		  (SELECT COUNT(*) FROM ls)::int                                                     AS list_count,
		  COUNT(*)::int                                                                      AS total_tasks,
		  COUNT(*) FILTER (WHERE completed_at IS NULL)::int                                  AS open_tasks,
		  COUNT(*) FILTER (WHERE completed_at IS NOT NULL)::int                              AS completed_tasks,
		  COUNT(*) FILTER (WHERE completed_at IS NULL AND due_at IS NOT NULL AND due_at < NOW())::int AS overdue_tasks,
		  COUNT(*) FILTER (WHERE completed_at IS NULL AND due_at IS NOT NULL
		                     AND due_at >= NOW() AND due_at < NOW() + INTERVAL '7 days')::int AS due_this_week
		  FROM ts`
	row := r.pool.QueryRow(ctx, q, portfolioID)
	var rollup domain.PortfolioRollup
	rollup.PortfolioID = portfolioID
	if err := row.Scan(
		&rollup.SpaceCount, &rollup.ListCount,
		&rollup.TotalTasks, &rollup.OpenTasks, &rollup.CompletedTasks,
		&rollup.OverdueTasks, &rollup.DueThisWeek,
	); err != nil {
		return nil, err
	}
	return &rollup, nil
}

type rowScanner interface{ Scan(...any) error }

func scanRow(r rowScanner) (*domain.Portfolio, error) {
	var p domain.Portfolio
	if err := r.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Description, &p.Color,
		&p.OwnerID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}
