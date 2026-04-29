package report

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) AccomplishedTasks(ctx context.Context, f domain.AccomplishmentFilter) ([]domain.AccomplishmentTask, error) {
	// "My accomplishments" = tasks completed in the window where I am the
	// creator or explicitly assigned. The DISTINCT guards against double rows
	// for tasks where the caller is both creator and assignee.
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT t.id, t.name, t.list_id, l.name AS list_name, t.archived, t.completed_at
		FROM tasks t
		JOIN lists l  ON l.id = t.list_id
		JOIN spaces s ON s.id = l.space_id
		LEFT JOIN task_assignees ta ON ta.task_id = t.id AND ta.user_id = $2
		WHERE s.workspace_id = $1
		  AND (t.creator_id = $2 OR ta.user_id = $2)
		  AND t.completed_at IS NOT NULL
		  AND t.completed_at >= $3
		  AND t.completed_at <  $4
		ORDER BY t.completed_at DESC
	`, f.WorkspaceID, f.UserID, f.From, f.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.AccomplishmentTask
	for rows.Next() {
		var t domain.AccomplishmentTask
		if err := rows.Scan(&t.ID, &t.Name, &t.ListID, &t.ListName, &t.Archived, &t.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
