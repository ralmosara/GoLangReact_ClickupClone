package dashboard

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

const dashCols = `id, workspace_id, space_id, name, creator_id, created_at, updated_at`
const widgetCols = `id, dashboard_id, kind, title, config, order_index, created_at, updated_at`

func scanDashboard(row pgx.Row) (*domain.Dashboard, error) {
	var d domain.Dashboard
	err := row.Scan(&d.ID, &d.WorkspaceID, &d.SpaceID, &d.Name, &d.CreatorID, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repo) Create(ctx context.Context, d *domain.Dashboard) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO dashboards (workspace_id, space_id, name, creator_id)
		VALUES ($1,$2,$3,$4)
		RETURNING id, created_at, updated_at
	`, d.WorkspaceID, d.SpaceID, d.Name, d.CreatorID).
		Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Dashboard, error) {
	return scanDashboard(r.pool.QueryRow(ctx, `SELECT `+dashCols+` FROM dashboards WHERE id=$1`, id))
}

func (r *Repo) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Dashboard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+dashCols+` FROM dashboards WHERE workspace_id=$1 ORDER BY created_at`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Dashboard
	for rows.Next() {
		var d domain.Dashboard
		if err := rows.Scan(&d.ID, &d.WorkspaceID, &d.SpaceID, &d.Name, &d.CreatorID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, d *domain.Dashboard) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE dashboards SET name=$2, space_id=$3, updated_at=NOW() WHERE id=$1`,
		d.ID, d.Name, d.SpaceID)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM dashboards WHERE id=$1`, id)
	return err
}

/* --- widgets ------------------------------------------------------------- */

func (r *Repo) CreateWidget(ctx context.Context, w *domain.Widget) error {
	cfg := w.Config
	if len(cfg) == 0 {
		cfg = []byte("{}")
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO dashboard_widgets (dashboard_id, kind, title, config, order_index)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at, updated_at
	`, w.DashboardID, w.Kind, w.Title, cfg, w.OrderIndex).
		Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

func (r *Repo) GetWidget(ctx context.Context, id uuid.UUID) (*domain.Widget, error) {
	var w domain.Widget
	err := r.pool.QueryRow(ctx, `SELECT `+widgetCols+` FROM dashboard_widgets WHERE id=$1`, id).
		Scan(&w.ID, &w.DashboardID, &w.Kind, &w.Title, &w.Config, &w.OrderIndex, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *Repo) ListWidgets(ctx context.Context, dashboardID uuid.UUID) ([]domain.Widget, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+widgetCols+` FROM dashboard_widgets WHERE dashboard_id=$1 ORDER BY order_index, created_at`,
		dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Widget
	for rows.Next() {
		var w domain.Widget
		if err := rows.Scan(&w.ID, &w.DashboardID, &w.Kind, &w.Title, &w.Config, &w.OrderIndex, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repo) UpdateWidget(ctx context.Context, w *domain.Widget) error {
	cfg := w.Config
	if len(cfg) == 0 {
		cfg = []byte("{}")
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE dashboard_widgets SET kind=$2, title=$3, config=$4, order_index=$5, updated_at=NOW()
		WHERE id=$1
	`, w.ID, w.Kind, w.Title, cfg, w.OrderIndex)
	return err
}

func (r *Repo) DeleteWidget(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM dashboard_widgets WHERE id=$1`, id)
	return err
}
