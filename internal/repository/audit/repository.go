package audit

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const cols = `id, workspace_id, actor_id, entity_type, entity_id, verb, before_json, after_json, created_at`

func (r *Repo) Create(ctx context.Context, e *domain.AuditEntry) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO audit_log (workspace_id, actor_id, entity_type, entity_id, verb, before_json, after_json)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at
	`, e.WorkspaceID, e.ActorID, e.EntityType, e.EntityID, e.Verb, e.BeforeJSON, e.AfterJSON).Scan(&e.ID, &e.CreatedAt)
}

func (r *Repo) List(ctx context.Context, f domain.AuditFilter) ([]domain.AuditEntry, error) {
	var where []string
	var args []any
	i := 1
	add := func(clause string, v any) {
		where = append(where, strings.ReplaceAll(clause, "?", "$"+strconv.Itoa(i)))
		args = append(args, v)
		i++
	}
	if f.WorkspaceID != nil {
		add("workspace_id = ?", *f.WorkspaceID)
	}
	if f.EntityType != nil {
		add("entity_type = ?", *f.EntityType)
	}
	if f.EntityID != nil {
		add("entity_id = ?", *f.EntityID)
	}
	if f.Before != nil {
		add("created_at < ?", *f.Before)
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	q := `SELECT ` + cols + ` FROM audit_log`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(i)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AuditEntry
	for rows.Next() {
		var e domain.AuditEntry
		if err := rows.Scan(&e.ID, &e.WorkspaceID, &e.ActorID, &e.EntityType, &e.EntityID, &e.Verb, &e.BeforeJSON, &e.AfterJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
