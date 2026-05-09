package search

import (
	"context"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// buildPrefixTSQuery turns a free-form user query into a tsquery string with
// prefix matching on every term, e.g. "auth tok" -> "auth:* & tok:*".
//
// We tokenize on any non-alphanumeric/underscore run, which also strips every
// character that has special meaning in tsquery syntax (& | ! ( ) : * < ' " etc.),
// so the result is always safe to pass to to_tsquery without further escaping.
// Returns "" if the input has no usable tokens — callers should short-circuit
// in that case rather than executing the SQL.
func buildPrefixTSQuery(raw string) string {
	lowered := strings.ToLower(raw)
	tokens := strings.FieldsFunc(lowered, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
	})
	if len(tokens) == 0 {
		return ""
	}
	parts := make([]string, len(tokens))
	for i, t := range tokens {
		parts[i] = t + ":*"
	}
	return strings.Join(parts, " & ")
}

// Search runs a plain-language query against all searchable entities within the
// given workspace. The user query is tokenized into a prefix tsquery
// (e.g. "auth tok" -> "auth:* & tok:*") so partial words still hit; this is
// what users almost always expect from a typeahead palette. Each entity is
// scoped to the workspace through joins on the hierarchy.
func (r *Repo) Search(ctx context.Context, workspaceID uuid.UUID, query string, limit int) ([]domain.SearchHit, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	tsq := buildPrefixTSQuery(query)
	if tsq == "" {
		// Nothing tokenizable (empty / pure punctuation) — no point hitting the DB.
		// The handler coerces a nil slice to [] for the JSON response.
		return nil, nil
	}
	q := `
		WITH q AS (SELECT to_tsquery('simple', $2) AS tsq)
		SELECT * FROM (
			-- Tasks scoped to this workspace through spaces→lists→tasks.
			SELECT 'task'::text AS entity_type,
			       t.id         AS entity_id,
			       t.name       AS title,
			       ts_headline('simple', coalesce(t.description, ''), (SELECT tsq FROM q),
			                   'MaxFragments=1,MaxWords=20,MinWords=5,ShortWord=2,HighlightAll=FALSE') AS snippet,
			       ts_rank(t.search_tsv, (SELECT tsq FROM q))::float8 AS rank,
			       $1::uuid   AS workspace_id,
			       t.list_id  AS list_id,
			       t.id       AS task_id,
			       NULL::uuid AS doc_id,
			       NULL::uuid AS channel_id,
			       t.created_at
			FROM tasks t
			JOIN lists l  ON l.id = t.list_id
			JOIN spaces s ON s.id = l.space_id
			WHERE s.workspace_id = $1
			  AND t.search_tsv @@ (SELECT tsq FROM q)

			UNION ALL

			SELECT 'doc',
			       d.id,
			       d.title,
			       ts_headline('simple', coalesce(d.content_text, ''), (SELECT tsq FROM q),
			                   'MaxFragments=1,MaxWords=20,MinWords=5,ShortWord=2,HighlightAll=FALSE'),
			       ts_rank(d.search_tsv, (SELECT tsq FROM q))::float8,
			       d.workspace_id,
			       NULL, NULL, d.id, NULL,
			       d.created_at
			FROM docs d
			WHERE d.workspace_id = $1 AND d.archived = FALSE
			  AND d.search_tsv @@ (SELECT tsq FROM q)

			UNION ALL

			SELECT 'comment',
			       c.id,
			       'Comment on task',
			       ts_headline('simple', coalesce(c.body, ''), (SELECT tsq FROM q),
			                   'MaxFragments=1,MaxWords=20,MinWords=5,ShortWord=2,HighlightAll=FALSE'),
			       ts_rank(c.search_tsv, (SELECT tsq FROM q))::float8,
			       $1, t2.list_id, c.task_id, NULL, NULL,
			       c.created_at
			FROM comments c
			JOIN tasks t2 ON t2.id = c.task_id
			JOIN lists l2 ON l2.id = t2.list_id
			JOIN spaces s2 ON s2.id = l2.space_id
			WHERE s2.workspace_id = $1
			  AND c.search_tsv @@ (SELECT tsq FROM q)

			UNION ALL

			SELECT 'message',
			       m.id,
			       coalesce(ch.name, 'Channel message'),
			       ts_headline('simple', coalesce(m.body, ''), (SELECT tsq FROM q),
			                   'MaxFragments=1,MaxWords=20,MinWords=5,ShortWord=2,HighlightAll=FALSE'),
			       ts_rank(m.search_tsv, (SELECT tsq FROM q))::float8,
			       ch.workspace_id,
			       NULL, NULL, NULL, m.channel_id,
			       m.created_at
			FROM messages m
			JOIN channels ch ON ch.id = m.channel_id
			WHERE ch.workspace_id = $1 AND m.deleted_at IS NULL
			  AND m.search_tsv @@ (SELECT tsq FROM q)
		) hits
		ORDER BY rank DESC, created_at DESC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, workspaceID, tsq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SearchHit
	for rows.Next() {
		var h domain.SearchHit
		if err := rows.Scan(
			&h.EntityType, &h.EntityID, &h.Title, &h.Snippet, &h.Rank,
			&h.WorkspaceID, &h.ListID, &h.TaskID, &h.DocID, &h.ChannelID, &h.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
