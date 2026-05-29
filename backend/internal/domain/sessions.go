package domain

import (
	"time"

	"github.com/google/uuid"
)

type Sessions struct {
	Id        uuid.UUID  `db:"id" json:"id"`
	UserId    uuid.UUID  `db:"user_id" json:"user_id"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	ExpiresAt time.Time  `db:"expires_at" json:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at" json:"revoked_at"`
}

type CreateSessionInput struct {
	UserId    uuid.UUID
	ExpiresAt time.Time
}
