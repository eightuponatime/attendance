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
		select id, google_sub, email, password_hash,
			last_name, first_name, middle_name, full_name, created_at
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
		select id, google_sub, email, password_hash,
			last_name, first_name, middle_name, full_name, created_at
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

func (r *UsersRepository) GetByEmail(ctx context.Context, email string) (*domain.Users, error) {
	const query = `
		select id, google_sub, email, password_hash,
			last_name, first_name, middle_name, full_name, created_at
		from users
		where email = $1
	`

	q := extractTransaction(ctx, r.db)
	var user domain.Users
	if err := sqlx.GetContext(ctx, q, &user, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *UsersRepository) CreateLocal(
	ctx context.Context,
	input domain.LocalRegisterInput,
	passwordHash string,
	fullName string,
) (*domain.Users, error) {
	const query = `
		insert into users (
			email, password_hash, last_name, first_name, middle_name, full_name
		)
		values ($1, $2, $3, $4, nullif($5, ''), $6)
		returning id, google_sub, email, password_hash,
			last_name, first_name, middle_name, full_name, created_at
	`

	q := extractTransaction(ctx, r.db)
	var user domain.Users
	if err := sqlx.GetContext(
		ctx,
		q,
		&user,
		query,
		input.Email,
		passwordHash,
		input.LastName,
		input.FirstName,
		input.MiddleName,
		fullName,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UsersRepository) LinkGoogleSub(
	ctx context.Context,
	userId uuid.UUID,
	googleSub string,
) (*domain.Users, error) {
	const query = `
		update users
		set google_sub = $2
		where id = $1
		returning id, google_sub, email, password_hash,
			last_name, first_name, middle_name, full_name, created_at
	`

	q := extractTransaction(ctx, r.db)
	var user domain.Users
	if err := sqlx.GetContext(ctx, q, &user, query, userId, googleSub); err != nil {
		return nil, err
	}

	return &user, nil
}
