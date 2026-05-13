package domain

import (
	"time"

	"github.com/google/uuid"
)

type Users struct {
	Id        uuid.UUID `db:"id"`
	GoogleSub string    `db:"google_sub"`
	Email     string    `db:"email"`
	FullName  string    `db:"full_name"`
	CreatedAt time.Time `db:"created_at"`
}

type Sessions struct {
	Id        uuid.UUID `db:"id"`
	UserId    uuid.UUID `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}

type AttendanceRecords struct {
	Id           uuid.UUID  `db:"id"`
	UserId       uuid.UUID  `db:"user_id"`
	BusinessDate time.Time  `db:"business_date"`
	CheckInAt    time.Time  `db:"check_in_at"`
	CheckOutAt   *time.Time `db:"check_out_at"`
	CreatedAt    time.Time  `db:"created_at"`
}

type SystemHeartbeat struct {
	Id        uuid.UUID `db:"id"`
	StartedAt time.Time `db:"started_at"`
	EndedAt   time.Time `db:"ended_at"`
	Reason    *string   `db:"reason"`
	CreatedAt time.Time `db:"created_at"`
}
