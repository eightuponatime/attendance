package postgres

import (
	"context"
	"database/sql"
	"errors"

	"attendance/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type UsersRepository struct {
	db *sqlx.DB
}

func NewUsersRepository(db *sqlx.DB) *UsersRepository {
	return &UsersRepository{
		db: db,
	}
}

func (r *UsersRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Users, error) {
	const query = `
		select id, google_sub, email, full_name, created_at
		from users
		where id = $1
	`

	q := extractTransaction(ctx, r.db)
	var user domain.Users
	if err := sqlx.GetContext(ctx, q, &user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *UsersRepository) GetByGoogleSub(ctx context.Context, googleSub string) (*domain.Users, error) {
	const query = `
		select id, google_sub, email, full_name, created_at
		from users
		where google_sub = $1
	`

	q := extractTransaction(ctx, r.db)
	var user domain.Users
	if err := sqlx.GetContext(ctx, q, &user, query, googleSub); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *UsersRepository) CreateOrUpdateFromGoogle(
	ctx context.Context,
	input domain.GoogleUserInput,
) (*domain.Users, error) {
	const query = `
		insert into users (google_sub, email, full_name)
		values ($1, $2, $3)
		on conflict (google_sub) do update set
			email = excluded.email,
			full_name = excluded.full_name
		returning id, google_sub, email, full_name, created_at
	`

	q := extractTransaction(ctx, r.db)
	var user domain.Users
	if err := sqlx.GetContext(
		ctx, q, &user, query, input.GoogleSub, input.Email, input.FullName,
	); err != nil {
		return nil, err
	}

	return &user, nil
}
