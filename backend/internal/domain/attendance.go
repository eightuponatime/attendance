package domain

import (
	"time"

	"github.com/google/uuid"
)

type AttendanceRecords struct {
	Id           uuid.UUID `db:"id" json:"id"`
	UserId       uuid.UUID `db:"user_id" json:"user_id"`
	BusinessDate time.Time `db:"business_date" json:"business_date"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type AttendanceEvents struct {
	Id         uuid.UUID `db:"id" json:"id"`
	RecordId   uuid.UUID `db:"record_id" json:"record_id"`
	EventType  string    `db:"event_type" json:"event_type"`
	EventAt    time.Time `db:"event_at" json:"event_at"`
	Status     string    `db:"status" json:"status"`
	PhoneModel string    `db:"phone_model" json:"phone_model"`
	Browser    string    `db:"browser" json:"browser"`
	DeviceId   string    `db:"device_id" json:"device_id"`
	ExternalIp string    `db:"external_ip" json:"external_ip"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type CreateAttendanceEventInput struct {
	RecordId   uuid.UUID
	EventType  string
	Status     string
	PhoneModel string
	Browser    string
	DeviceId   string
	ExternalIp string
}

type AttendanceMarkInput struct {
	UserId     uuid.UUID
	PhoneModel string
	Browser    string
	DeviceId   string
	ExternalIp string
}

type AttendanceToday struct {
	BusinessDate      time.Time          `json:"business_date"`
	Record            *AttendanceRecords `json:"record"`
	CheckIn           *AttendanceEvents  `json:"check_in"`
	CheckOut          *AttendanceEvents  `json:"check_out"`
	LateMinutes       int                `json:"late_minutes"`
	EarlyLeaveMinutes int                `json:"early_leave_minutes"`
	ImpactedByOutage  bool               `json:"impacted_by_outage"`
}

type AttendanceRangeEventRow struct {
	BusinessDate time.Time  `db:"business_date"`
	CheckInAt    *time.Time `db:"check_in_at"`
	CheckOutAt   *time.Time `db:"check_out_at"`
}

type AttendanceDaySummary struct {
	Date              time.Time
	CheckInAt         *time.Time
	CheckOutAt        *time.Time
	WorkedMinutes     int
	LateMinutes       int
	EarlyLeaveMinutes int
	Status            string
	ImpactedByOutage  bool
}

type AttendanceSummary struct {
	From                time.Time
	To                  time.Time
	WorkdayStart        string
	WorkdayEnd          string
	TargetMinutesPerDay int
	Days                []AttendanceDaySummary
}
