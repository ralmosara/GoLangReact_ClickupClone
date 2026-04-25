package message

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/clickup/internal/domain"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const cols = `id, channel_id, author_id, parent_message_id, body, edited_at, deleted_at, created_at`

func scan(row pgx.Row) (*domain.Message, error) {
	var m domain.Message
	err := row.Scan(&m.ID, &m.ChannelID, &m.AuthorID, &m.ParentMessageID, &m.Body, &m.EditedAt, &m.DeletedAt, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repo) Create(ctx context.Context, m *domain.Message) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO messages (channel_id, author_id, parent_message_id, body)
		VALUES ($1,$2,$3,$4)
		RETURNING id, created_at
	`, m.ChannelID, m.AuthorID, m.ParentMessageID, m.Body).Scan(&m.ID, &m.CreatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM messages WHERE id=$1`, id))
}

func (r *Repo) List(ctx context.Context, f domain.MessageFilter) ([]domain.Message, error) {
	var where []string
	var args []any
	i := 1
	add := func(clause string, v any) {
		where = append(where, strings.ReplaceAll(clause, "?", "$"+strconv.Itoa(i)))
		args = append(args, v)
		i++
	}
	add("channel_id = ?", f.ChannelID)
	if f.ParentID != nil {
		add("parent_message_id = ?", *f.ParentID)
	} else {
		where = append(where, "parent_message_id IS NULL")
	}
	if f.Before != nil {
		add("created_at < ?", *f.Before)
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := `SELECT ` + cols + ` FROM messages WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(i)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Message
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.AuthorID, &m.ParentMessageID, &m.Body,
			&m.EditedAt, &m.DeletedAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, m *domain.Message) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE messages SET body=$2, edited_at=NOW() WHERE id=$1`, m.ID, m.Body)
	return err
}

func (r *Repo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE messages SET deleted_at=NOW(), body='' WHERE id=$1`, id)
	return err
}

func (r *Repo) CountThreadReplies(ctx context.Context, parentID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM messages WHERE parent_message_id=$1 AND deleted_at IS NULL`,
		parentID).Scan(&n)
	return n, err
}
