package service

import (
	"context"
	"time"

	"attendance/internal/domain"

	"github.com/google/uuid"
)

type UsersService interface {
	FindOrCreateFromGoogle(ctx context.Context, input domain.GoogleUserInput) (*domain.Users, error)
	RegisterLocal(ctx context.Context, input domain.LocalRegisterInput) (*domain.Users, error)
	LoginLocal(ctx context.Context, input domain.LocalLoginInput) (*domain.Users, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Users, error)
	GetByGoogleSub(ctx context.Context, googleSub string) (*domain.Users, error)
}

type SessionsService interface {
	Create(ctx context.Context, userId uuid.UUID) (*domain.Sessions, error)
	GetValidByID(ctx context.Context, id uuid.UUID) (*domain.Sessions, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type AttendanceService interface {
	Today(ctx context.Context, userId uuid.UUID) (*domain.AttendanceToday, error)
	CheckIn(ctx context.Context, input domain.AttendanceMarkInput) (*domain.AttendanceToday, error)
	CheckOut(ctx context.Context, input domain.AttendanceMarkInput) (*domain.AttendanceToday, error)
	Summary(ctx context.Context, userId uuid.UUID, from time.Time, to time.Time) (*domain.AttendanceSummary, error)
	SubmitExplanation(ctx context.Context, input domain.CreateAttendanceExplanationInput) (*domain.AttendanceExplanation, error)
}

type SystemService interface {
	ImpactedBusinessDates(ctx context.Context, from time.Time, to time.Time) ([]time.Time, error)
}

type AdminService interface {
	IsAdmin(ctx context.Context, email string) (bool, error)
	CreateSession(ctx context.Context, input domain.GoogleUserInput) (*domain.AdminSession, error)
	GetValidSessionByID(ctx context.Context, id uuid.UUID) (*domain.AdminSession, error)
	ListAccess(ctx context.Context) ([]domain.AdminAccess, error)
	AddAccess(ctx context.Context, input domain.CreateAdminAccessInput) (*domain.AdminAccess, error)
	RevokeAccess(ctx context.Context, actorEmail string, email string) error
	ListReports(ctx context.Context) ([]domain.AdminReportRun, error)
	GetReportRunByPeriod(ctx context.Context, from time.Time, to time.Time) (*domain.AdminReportRun, error)
	CreateReportRun(ctx context.Context, input domain.CreateAdminReportRunInput) (*domain.AdminReportRun, error)
	ListSessions(ctx context.Context) ([]domain.AdminSession, error)
	RevokeSession(ctx context.Context, sessionId uuid.UUID) error
	EmployeesMonth(ctx context.Context, from time.Time, to time.Time) (*domain.AdminEmployeesMonthOverview, error)
	EmployeeMonth(ctx context.Context, userId uuid.UUID, from time.Time, to time.Time) (*domain.AdminEmployeeMonthSummary, error)
	SuspiciousActivity(ctx context.Context, from time.Time, to time.Time) (*domain.AdminSuspiciousActivity, error)
	ListSystemOutages(ctx context.Context, from time.Time, to time.Time) ([]domain.SystemOutage, error)
	OutageDayEmployees(ctx context.Context, outageId uuid.UUID) (*domain.SystemOutage, []domain.AdminOutageDayEmployeeRow, error)
	RepairOutageDay(ctx context.Context, input domain.AdminOutageRepairInput) error
	ListExplanations(ctx context.Context, from time.Time, to time.Time, status string) ([]domain.AdminExplanationRow, error)
	ApproveExplanation(ctx context.Context, input domain.AdminExplanationDecisionInput) error
	RejectExplanation(ctx context.Context, input domain.AdminExplanationDecisionInput) error
}
