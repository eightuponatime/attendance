package domain

import (
	"time"

	"github.com/google/uuid"
)

type Users struct {
	Id        uuid.UUID `db:"id" json:"id"`
	GoogleSub string    `db:"google_sub" json:"google_sub"`
	Email     string    `db:"email" json:"email"`
	FullName  string    `db:"full_name" json:"full_name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type GoogleUserInput struct {
	GoogleSub string
	Email     string
	FullName  string
}
