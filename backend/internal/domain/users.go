package domain

import (
	"time"

	"github.com/google/uuid"
)

type Users struct {
	Id           uuid.UUID `db:"id" json:"id"`
	GoogleSub    *string   `db:"google_sub" json:"google_sub"`
	Email        string    `db:"email" json:"email"`
	PasswordHash *string   `db:"password_hash" json:"-"`
	LastName     string    `db:"last_name" json:"last_name"`
	FirstName    string    `db:"first_name" json:"first_name"`
	MiddleName   *string   `db:"middle_name" json:"middle_name"`
	FullName     string    `db:"full_name" json:"full_name"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type GoogleUserInput struct {
	GoogleSub string
	Email     string
	FullName  string
}

type LocalRegisterInput struct {
	Email      string
	Password   string
	LastName   string
	FirstName  string
	MiddleName string
	GoogleSub  string
}

type LocalLoginInput struct {
	Email    string
	Password string
}
