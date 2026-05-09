package comment

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

// scanCols is the canonical select-list. Always returns the threading
// + soft-delete columns so a single scan can populate the full struct.
const scanCols = `
    id, task_id, author_id, parent_comment_id, body,
    created_at, updated_at, edited_at, deleted_at, deleted_by`

func (r *Repo) Create(ctx context.Context, c *domain.Comment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO comments (task_id, author_id, parent_comment_id, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, c.TaskID, c.AuthorID, c.ParentCommentID, c.Body).
		Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// ListByTask returns top-level comments only — replies are loaded
// lazily per-parent. ReplyCount is computed in-query so the UI can
// render "3 replies ↪" without a follow-up roundtrip.
func (r *Repo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+scanCols+`,
		       (SELECT COUNT(*) FROM comments c2
		         WHERE c2.parent_comment_id = comments.id
		           AND c2.deleted_at IS NULL) AS reply_count
		  FROM comments
		 WHERE task_id = $1
		   AND parent_comment_id IS NULL
		 ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanList(rows, true)
}

func (r *Repo) ListReplies(ctx context.Context, parentID uuid.UUID) ([]domain.Comment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+scanCols+`, 0 AS reply_count
		  FROM comments
		 WHERE parent_comment_id = $1
		 ORDER BY created_at ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanList(rows, true)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+scanCols+`, 0 AS reply_count FROM comments WHERE id = $1
	`, id)
	c, err := scanRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *Repo) UpdateBody(ctx context.Context, id uuid.UUID, body string) (*domain.Comment, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE comments
		   SET body       = $2,
		       edited_at  = NOW(),
		       updated_at = NOW()
		 WHERE id = $1
		RETURNING `+scanCols+`, 0 AS reply_count
	`, id, body)
	return scanRow(row)
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM comments WHERE id = $1`, id)
	return err
}

func (r *Repo) SoftDelete(ctx context.Context, id, actor uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE comments
		   SET deleted_at = NOW(),
		       deleted_by = $2
		 WHERE id = $1
	`, id, actor)
	return err
}

// AddReaction is idempotent — DO NOTHING on the (comment, user, emoji)
// PK conflict so double-tapping the same emoji isn't an error.
func (r *Repo) AddReaction(ctx context.Context, commentID, userID uuid.UUID, emoji string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO comment_reactions (comment_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, commentID, userID, emoji)
	return err
}

func (r *Repo) RemoveReaction(ctx context.Context, commentID, userID uuid.UUID, emoji string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM comment_reactions
		 WHERE comment_id = $1 AND user_id = $2 AND emoji = $3
	`, commentID, userID, emoji)
	return err
}

// ListReactionSummaries does the per-comment aggregation in a single
// pass. We project bool_or(user_id = viewer) so each emoji bucket
// also tells the UI whether the caller has reacted — saves a second
// query just to render the "you reacted" indicator.
func (r *Repo) ListReactionSummaries(ctx context.Context, commentIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]domain.CommentReactionSummary, error) {
	out := make(map[uuid.UUID][]domain.CommentReactionSummary, len(commentIDs))
	if len(commentIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT comment_id,
		       emoji,
		       COUNT(*)::int                         AS cnt,
		       BOOL_OR(user_id = $2)                 AS reacted
		  FROM comment_reactions
		 WHERE comment_id = ANY($1::uuid[])
		 GROUP BY comment_id, emoji
		 ORDER BY comment_id, MIN(created_at)
	`, commentIDs, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid     uuid.UUID
			summary domain.CommentReactionSummary
		)
		if err := rows.Scan(&cid, &summary.Emoji, &summary.Count, &summary.Reacted); err != nil {
			return nil, err
		}
		out[cid] = append(out[cid], summary)
	}
	return out, rows.Err()
}

// scanRow / scanList centralise the field order so the canonical select
// can grow a column without every call site needing to track it.

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRow(r rowScanner) (*domain.Comment, error) {
	var (
		c            domain.Comment
		replyCount   int
	)
	if err := r.Scan(
		&c.ID, &c.TaskID, &c.AuthorID, &c.ParentCommentID, &c.Body,
		&c.CreatedAt, &c.UpdatedAt, &c.EditedAt, &c.DeletedAt, &c.DeletedBy,
		&replyCount,
	); err != nil {
		return nil, err
	}
	c.ReplyCount = replyCount
	return &c, nil
}

func scanList(rows pgx.Rows, _ bool) ([]domain.Comment, error) {
	var out []domain.Comment
	for rows.Next() {
		c, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}
