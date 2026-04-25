package channel

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

const cols = `id, workspace_id, space_id, name, topic, kind, is_private, creator_id, created_at, updated_at`

func scan(row pgx.Row) (*domain.Channel, error) {
	var c domain.Channel
	err := row.Scan(&c.ID, &c.WorkspaceID, &c.SpaceID, &c.Name, &c.Topic, &c.Kind, &c.IsPrivate,
		&c.CreatorID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanMany(rows pgx.Rows) ([]domain.Channel, error) {
	var out []domain.Channel
	for rows.Next() {
		var c domain.Channel
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.SpaceID, &c.Name, &c.Topic, &c.Kind, &c.IsPrivate,
			&c.CreatorID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, c *domain.Channel) error {
	kind := c.Kind
	if kind == "" {
		kind = domain.ChannelKindChannel
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO channels (workspace_id, space_id, name, topic, kind, is_private, creator_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at
	`, c.WorkspaceID, c.SpaceID, c.Name, c.Topic, kind, c.IsPrivate, c.CreatorID).
		Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Channel, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM channels WHERE id=$1`, id))
}

func (r *Repo) ListByWorkspace(ctx context.Context, wsID uuid.UUID) ([]domain.Channel, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM channels WHERE workspace_id=$1 ORDER BY created_at`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]domain.Channel, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+cols+` FROM channels WHERE space_id=$1 ORDER BY created_at`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMany(rows)
}

func (r *Repo) Update(ctx context.Context, c *domain.Channel) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels SET name=$2, topic=$3, is_private=$4, updated_at=NOW() WHERE id=$1`,
		c.ID, c.Name, c.Topic, c.IsPrivate)
	return err
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channels WHERE id=$1`, id)
	return err
}

func (r *Repo) AddMember(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO channel_members (channel_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
		channelID, userID)
	return err
}

func (r *Repo) RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM channel_members WHERE channel_id=$1 AND user_id=$2`,
		channelID, userID)
	return err
}

func (r *Repo) ListMembers(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM channel_members WHERE channel_id=$1 ORDER BY joined_at`,
		channelID)
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

func (r *Repo) TouchRead(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channel_members SET last_read_at = NOW() WHERE channel_id=$1 AND user_id=$2`,
		channelID, userID)
	return err
}
