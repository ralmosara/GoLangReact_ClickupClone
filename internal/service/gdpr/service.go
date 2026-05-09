// Package gdpr ships the workspace data-export endpoint required for
// Article 15 (right of access) and Article 20 (right to data portability).
// The export is a JSON document containing every row a workspace owns
// (tasks, comments, time entries, audit log, etc.) — sized for human
// review, not for re-import.
//
// The right-to-be-forgotten endpoint (Article 17) is intentionally NOT in
// this package yet — cascading workspace deletion across 30+ tables needs
// careful per-table handling and is on the deferred list for the next
// iteration of phase 2.
package gdpr

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// ExportTable is the on-the-wire shape: a table name plus an array of rows
// represented as JSON objects. Streamed serially rather than nested to
// keep the encoder happy on large workspaces.
type ExportTable struct {
	Table string            `json:"table"`
	Rows  []json.RawMessage `json:"rows"`
}

// Tables to include. Each query is workspace-scoped — the SQL must filter
// to wsID either directly (rows that have workspace_id) or transitively
// (rows that join through spaces / lists / tasks).
//
// Order matters for human readability: identity first, then the org
// hierarchy, then the leaf data.
var exportQueries = []struct {
	name string
	sql  string
}{
	{"workspaces", `SELECT row_to_json(t) FROM workspaces t WHERE id = $1`},
	{"workspace_members", `SELECT row_to_json(t) FROM workspace_members t WHERE workspace_id = $1`},
	{"spaces", `SELECT row_to_json(t) FROM spaces t WHERE workspace_id = $1`},
	{"folders", `SELECT row_to_json(t) FROM folders t
	             WHERE space_id IN (SELECT id FROM spaces WHERE workspace_id = $1)`},
	{"lists", `SELECT row_to_json(t) FROM lists t
	           WHERE space_id IN (SELECT id FROM spaces WHERE workspace_id = $1)`},
	{"statuses", `SELECT row_to_json(t) FROM statuses t WHERE workspace_id = $1`},
	{"tags", `SELECT row_to_json(t) FROM tags t WHERE workspace_id = $1`},
	{"tasks", `SELECT row_to_json(t) FROM tasks t
	           WHERE list_id IN (SELECT l.id FROM lists l JOIN spaces s ON s.id = l.space_id WHERE s.workspace_id = $1)`},
	{"task_assignees", `SELECT row_to_json(ta) FROM task_assignees ta
	                    JOIN tasks t ON t.id = ta.task_id
	                    JOIN lists l ON l.id = t.list_id
	                    JOIN spaces s ON s.id = l.space_id
	                    WHERE s.workspace_id = $1`},
	{"comments", `SELECT row_to_json(c) FROM comments c
	              JOIN tasks t ON t.id = c.task_id
	              JOIN lists l ON l.id = t.list_id
	              JOIN spaces s ON s.id = l.space_id
	              WHERE s.workspace_id = $1`},
	{"attachments", `SELECT row_to_json(a) FROM attachments a
	                 JOIN tasks t ON t.id = a.task_id
	                 JOIN lists l ON l.id = t.list_id
	                 JOIN spaces s ON s.id = l.space_id
	                 WHERE s.workspace_id = $1`},
	{"time_entries", `SELECT row_to_json(te) FROM time_entries te
	                  JOIN tasks t ON t.id = te.task_id
	                  JOIN lists l ON l.id = t.list_id
	                  JOIN spaces s ON s.id = l.space_id
	                  WHERE s.workspace_id = $1`},
	{"docs", `SELECT row_to_json(t) FROM docs t WHERE workspace_id = $1`},
	{"goals", `SELECT row_to_json(t) FROM goals t WHERE workspace_id = $1`},
	{"sprints", `SELECT row_to_json(sp) FROM sprints sp
	             JOIN lists l ON l.id = sp.list_id
	             JOIN spaces s ON s.id = l.space_id
	             WHERE s.workspace_id = $1`},
	{"automations", `SELECT row_to_json(t) FROM automations t WHERE workspace_id = $1`},
	{"channels", `SELECT row_to_json(t) FROM channels t WHERE workspace_id = $1`},
	{"audit_log", `SELECT row_to_json(t) FROM audit_log t WHERE workspace_id = $1 ORDER BY created_at`},
}

// Export collects the workspace's rows table-by-table. The result is held
// in memory — workable for the workspaces this product targets (low six
// figures of rows max), and we explicitly cap each table at 100k rows
// below that to refuse pathological exports rather than OOM the server.
//
// For larger tenants this should switch to streaming JSON Lines via
// io.Writer; the handler shape would change but this service contract
// would not.
func (s *Service) Export(ctx context.Context, wsID uuid.UUID) ([]ExportTable, error) {
	const perTableCap = 100_000

	out := make([]ExportTable, 0, len(exportQueries))
	for _, q := range exportQueries {
		rows, err := s.pool.Query(ctx, q.sql, wsID)
		if err != nil {
			return nil, fmt.Errorf("gdpr export %s: %w", q.name, err)
		}
		var bucket []json.RawMessage
		for rows.Next() {
			if len(bucket) >= perTableCap {
				rows.Close()
				return nil, fmt.Errorf("gdpr export %s: exceeded per-table cap of %d rows", q.name, perTableCap)
			}
			var raw json.RawMessage
			if err := rows.Scan(&raw); err != nil {
				rows.Close()
				return nil, fmt.Errorf("gdpr export %s: scan: %w", q.name, err)
			}
			bucket = append(bucket, raw)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, fmt.Errorf("gdpr export %s: iter: %w", q.name, err)
		}
		out = append(out, ExportTable{Table: q.name, Rows: bucket})
	}
	return out, nil
}
