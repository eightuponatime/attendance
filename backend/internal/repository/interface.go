package repository

import (
	"context"
	"time"

	"attendance/internal/domain"

	"github.com/google/uuid"
)

type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type UsersRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Users, error)
	GetByGoogleSub(ctx context.Context, googleSub string) (*domain.Users, error)
	GetByEmail(ctx context.Context, email string) (*domain.Users, error)
	CreateLocal(ctx context.Context, input domain.LocalRegisterInput, passwordHash string, fullName string) (*domain.Users, error)
	LinkGoogleSub(ctx context.Context, userId uuid.UUID, googleSub string) (*domain.Users, error)
}

type SessionsRepository interface {
	Create(ctx context.Context, input domain.CreateSessionInput) (*domain.Sessions, error)
	GetValidByID(ctx context.Context, id uuid.UUID) (*domain.Sessions, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeActiveByUserID(ctx context.Context, userId uuid.UUID) error
}

type AttendanceRepository interface {
	GetRecordByUserAndDate(ctx context.Context, userId uuid.UUID, businessDate time.Time) (*domain.AttendanceRecords, error)
	CreateOrGetRecord(ctx context.Context, userId uuid.UUID, businessDate time.Time) (*domain.AttendanceRecords, error)
	GetEventByRecordAndType(ctx context.Context, recordId uuid.UUID, eventType string) (*domain.AttendanceEvents, error)
	CreateEvent(ctx context.Context, input domain.CreateAttendanceEventInput) (*domain.AttendanceEvents, error)
	GetRangeEventRows(ctx context.Context, userId uuid.UUID, from time.Time, to time.Time) ([]domain.AttendanceRangeEventRow, error)
}

type SystemRepository interface {
	ListImpactedBusinessDates(ctx context.Context, from time.Time, to time.Time) ([]time.Time, error)
}

type AdminRepository interface {
	IsActiveAdminByEmail(ctx context.Context, email string) (bool, error)
	ListAccess(ctx context.Context) ([]domain.AdminAccess, error)
	UpsertAccess(ctx context.Context, input domain.CreateAdminAccessInput) (*domain.AdminAccess, error)
	RevokeAccess(ctx context.Context, email string) error
	CreateSession(ctx context.Context, input domain.CreateAdminSessionInput) (*domain.AdminSession, error)
	GetValidSessionByID(ctx context.Context, id uuid.UUID) (*domain.AdminSession, error)
	ListReports(ctx context.Context) ([]domain.AdminReportRun, error)
	GetReportRunByPeriod(ctx context.Context, from time.Time, to time.Time) (*domain.AdminReportRun, error)
	CreateReportRun(ctx context.Context, input domain.CreateAdminReportRunInput) (*domain.AdminReportRun, error)
	ListSessions(ctx context.Context) ([]domain.AdminSession, error)
	RevokeSession(ctx context.Context, sessionId uuid.UUID) error
	RevokeSessionsByEmail(ctx context.Context, email string) error
	ListEmployeeMonthRows(ctx context.Context, from time.Time, to time.Time) ([]domain.AdminEmployeeMonthRow, error)
	ListEmployeeMonthRowsByUser(ctx context.Context, userId uuid.UUID, from time.Time, to time.Time) ([]domain.AdminEmployeeMonthRow, error)
	ListAttendanceEvents(ctx context.Context, from time.Time, to time.Time) ([]domain.AdminAttendanceEventRow, error)
	GetSystemHeartbeat(ctx context.Context) (*domain.SystemHeartbeat, error)
	UpdateSystemHeartbeat(ctx context.Context, seenAt time.Time) error
	CreateSystemOutage(ctx context.Context, input domain.CreateSystemOutageInput) (*domain.SystemOutage, error)
	ListSystemOutages(ctx context.Context, from time.Time, to time.Time) ([]domain.SystemOutage, error)
	GetSystemOutageByID(ctx context.Context, id uuid.UUID) (*domain.SystemOutage, error)
	ListOutageDayEmployees(ctx context.Context, businessDate time.Time) ([]domain.AdminOutageDayEmployeeRow, error)
	UpsertAttendanceEventAt(ctx context.Context, input domain.UpsertAttendanceEventAtInput) (*domain.UpsertAttendanceEventAtResult, error)
	CreateAttendanceAdjustment(ctx context.Context, input domain.CreateAttendanceAdjustmentInput) error
	ResolveSystemOutage(ctx context.Context, outageId uuid.UUID, adminEmail string, note string) error
}
