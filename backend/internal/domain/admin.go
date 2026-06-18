package domain

import (
	"time"

	"github.com/google/uuid"
)

type AdminAccess struct {
	Email     string     `db:"email" json:"email"`
	FullName  *string    `db:"full_name" json:"full_name"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	CreatedBy *string    `db:"created_by" json:"created_by"`
	RevokedAt *time.Time `db:"revoked_at" json:"revoked_at"`
}

type AdminSession struct {
	Id        uuid.UUID  `db:"id" json:"id"`
	Email     string     `db:"email" json:"email"`
	FullName  string     `db:"full_name" json:"full_name"`
	GoogleSub string     `db:"google_sub" json:"google_sub"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	ExpiresAt time.Time  `db:"expires_at" json:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at" json:"revoked_at"`
}

type CreateAdminAccessInput struct {
	Email     string
	CreatedBy string
}

type CreateAdminSessionInput struct {
	Email     string
	FullName  string
	GoogleSub string
	ExpiresAt time.Time
}

type AdminReportRun struct {
	Id          uuid.UUID `db:"id" json:"id"`
	PeriodStart time.Time `db:"period_start" json:"period_start"`
	PeriodEnd   time.Time `db:"period_end" json:"period_end"`
	SentAt      time.Time `db:"sent_at" json:"sent_at"`
}

type CreateAdminReportRunInput struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
}

type SystemHeartbeat struct {
	Id         int       `db:"id"`
	LastSeenAt time.Time `db:"last_seen_at"`
}

type SystemOutage struct {
	Id                   uuid.UUID  `db:"id"`
	StartedAt            time.Time  `db:"started_at"`
	EndedAt              time.Time  `db:"ended_at"`
	Reason               *string    `db:"reason"`
	CreatedAt            time.Time  `db:"created_at"`
	AffectedBusinessDate *time.Time `db:"affected_business_date"`
	ImpactsWorkHours     bool       `db:"impacts_work_hours"`
	ResolvedAt           *time.Time `db:"resolved_at"`
	ResolvedBy           *string    `db:"resolved_by"`
	ResolutionNote       *string    `db:"resolution_note"`
}

type CreateSystemOutageInput struct {
	StartedAt            time.Time
	EndedAt              time.Time
	Reason               string
	AffectedBusinessDate *time.Time
	ImpactsWorkHours     bool
}

type AdminOutageDayEmployeeRow struct {
	UserId     uuid.UUID  `db:"user_id"`
	Email      string     `db:"email"`
	FullName   string     `db:"full_name"`
	CheckInAt  *time.Time `db:"check_in_at"`
	CheckOutAt *time.Time `db:"check_out_at"`
}

type UpsertAttendanceEventAtInput struct {
	UserId       uuid.UUID
	BusinessDate time.Time
	EventType    string
	EventAt      time.Time
	Status       string
}

type UpsertAttendanceEventAtResult struct {
	OldEventAt *time.Time
	NewEventAt time.Time
}

type CreateAttendanceAdjustmentInput struct {
	UserId              uuid.UUID
	BusinessDate        time.Time
	EventType           string
	OldEventAt          *time.Time
	NewEventAt          time.Time
	Reason              string
	OutageId            uuid.UUID
	ExplanationId       uuid.UUID
	CreatedByAdminEmail string
	DecisionSource      string
}

type SetAttendanceEventAtInput struct {
	UserId       uuid.UUID
	BusinessDate time.Time
	EventType    string
	EventAt      *time.Time
	Status       string
}

type AdminOutageRepairItem struct {
	UserId     uuid.UUID
	CheckInAt  *time.Time
	CheckOutAt *time.Time
}

type AdminOutageRepairInput struct {
	OutageId       uuid.UUID
	AdminEmail     string
	ResolutionNote string
	Items          []AdminOutageRepairItem
}

type AdminEmployeeMonthRow struct {
	UserId        uuid.UUID  `db:"user_id"`
	Email         string     `db:"email"`
	FullName      string     `db:"full_name"`
	BusinessDate  *time.Time `db:"business_date"`
	CheckInAt     *time.Time `db:"check_in_at"`
	CheckOutAt    *time.Time `db:"check_out_at"`
	Voided        bool       `db:"voided"`
	VoidReason    *string    `db:"void_reason"`
	VoidedByAdmin *string    `db:"voided_by_admin"`
	VoidedAt      *time.Time `db:"voided_at"`
}

type AdminEmployeeMonthSummary struct {
	UserId           uuid.UUID
	Email            string
	FullName         string
	WorkedMinutes    int
	TargetMinutes    int
	CheckInCount     int
	CheckOutCount    int
	LateCount        int
	EarlyLeaveCount  int
	MissingCheckOuts int
	WorkedDays       int
	AttendanceDays   []AttendanceDaySummary
}

type AdminEmployeesMonthOverview struct {
	From                time.Time
	To                  time.Time
	WorkdayStart        string
	WorkdayEnd          string
	TargetMinutesPerDay int
	Employees           []AdminEmployeeMonthSummary
}

type AdminAttendanceEventRow struct {
	EventId      uuid.UUID `db:"event_id"`
	UserId       uuid.UUID `db:"user_id"`
	Email        string    `db:"email"`
	FullName     string    `db:"full_name"`
	BusinessDate time.Time `db:"business_date"`
	EventType    string    `db:"event_type"`
	EventAt      time.Time `db:"event_at"`
	DeviceId     string    `db:"device_id"`
	ExternalIp   string    `db:"external_ip"`
}

type AdminSuspiciousActivity struct {
	From          time.Time
	To            time.Time
	DeviceMatches []AdminSuspiciousDeviceMatch
	IPMatches     []AdminSuspiciousIPMatch
}

type AdminSuspiciousActor struct {
	UserId   uuid.UUID
	Email    string
	FullName string
}

type AdminSuspiciousDeviceMatch struct {
	DeviceId string
	Owner    AdminSuspiciousActor
	Event    AdminAttendanceEventRow
}

type AdminSuspiciousIPMatch struct {
	ExternalIp     string
	Event          AdminAttendanceEventRow
	PreviousEvent  AdminAttendanceEventRow
	MinutesBetween int
}

type AdminExplanationRow struct {
	AttendanceExplanation
	Email      string     `db:"email"`
	FullName   string     `db:"full_name"`
	CheckInAt  *time.Time `db:"check_in_at"`
	CheckOutAt *time.Time `db:"check_out_at"`
}

type AdminExplanationDecisionInput struct {
	ExplanationId uuid.UUID
	AdminEmail    string
	ReviewNote    string
	CheckInAt     *time.Time
	CheckOutAt    *time.Time
}

type AdminDayOverride struct {
	UserId               uuid.UUID  `db:"user_id"`
	BusinessDate         time.Time  `db:"business_date"`
	Status               string     `db:"status"`
	Reason               string     `db:"reason"`
	CreatedByAdminEmail  string     `db:"created_by_admin_email"`
	CreatedAt            time.Time  `db:"created_at"`
	RestoredByAdminEmail *string    `db:"restored_by_admin_email"`
	RestoredAt           *time.Time `db:"restored_at"`
	RestoreReason        *string    `db:"restore_reason"`
}

type AdminAuditLog struct {
	Id             uuid.UUID  `db:"id"`
	AdminEmail     string     `db:"admin_email"`
	UserId         *uuid.UUID `db:"user_id"`
	ExplanationId  *uuid.UUID `db:"explanation_id"`
	Email          *string    `db:"email"`
	FullName       *string    `db:"full_name"`
	BusinessDate   *time.Time `db:"business_date"`
	Action         string     `db:"action"`
	OldCheckInAt   *time.Time `db:"old_check_in_at"`
	OldCheckOutAt  *time.Time `db:"old_check_out_at"`
	NewCheckInAt   *time.Time `db:"new_check_in_at"`
	NewCheckOutAt  *time.Time `db:"new_check_out_at"`
	DecisionSource string     `db:"decision_source"`
	Reason         string     `db:"reason"`
	CreatedAt      time.Time  `db:"created_at"`
}

type AdminAuditInput struct {
	AdminEmail     string
	UserId         uuid.UUID
	ExplanationId  uuid.UUID
	BusinessDate   time.Time
	Action         string
	OldCheckInAt   *time.Time
	OldCheckOutAt  *time.Time
	NewCheckInAt   *time.Time
	NewCheckOutAt  *time.Time
	DecisionSource string
	Reason         string
}

type AdminVoidDayInput struct {
	UserId         uuid.UUID
	BusinessDate   time.Time
	AdminEmail     string
	Reason         string
	DecisionSource string
	ExplanationId  uuid.UUID
}
