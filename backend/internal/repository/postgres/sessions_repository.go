package postgres

import (
	"context"
	"database/sql"
	"errors"

	"attendance/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SessionsRepository struct {
	db *sqlx.DB
}

func NewSessionsRepository(db *sqlx.DB) *SessionsRepository {
	return &SessionsRepository{db: db}
}

func (r *SessionsRepository) Create(
	ctx context.Context,
	input domain.CreateSessionInput,
) (*domain.Sessions, error) {
	const query = `
		insert into sessions (user_id, expires_at)
		values ($1, $2)
		returning id, user_id, created_at, expires_at, revoked_at
	`

	q := extractTransaction(ctx, r.db)
	var session domain.Sessions
	if err := sqlx.GetContext(ctx, q, &session, query, input.UserId, input.ExpiresAt); err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *SessionsRepository) GetValidByID(ctx context.Context, id uuid.UUID) (*domain.Sessions, error) {
	const query = `
		select id, user_id, created_at, expires_at, revoked_at
		from sessions
		where id = $1
			and expires_at > now()
			and revoked_at is null
	`

	q := extractTransaction(ctx, r.db)
	var session domain.Sessions
	if err := sqlx.GetContext(ctx, q, &session, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &session, nil
}

func (r *SessionsRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	const query = `
		update sessions
		set revoked_at = now()
		where id = $1
			and revoked_at is null
	`

	q := extractTransaction(ctx, r.db)
	_, err := q.ExecContext(ctx, query, id)
	return err
}
