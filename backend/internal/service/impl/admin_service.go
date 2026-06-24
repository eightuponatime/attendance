package impl

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"attendance/config"
	"attendance/internal/domain"
	"attendance/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrInvalidAdminInput = errors.New("invalid admin input")
	ErrCannotRevokeSelf  = errors.New("cannot revoke your own admin access")
)

type AdminService struct {
	cfg       *config.Config
	txManager repository.TransactionManager
	rp        repository.AdminRepository
}

func NewAdminService(
	cfg *config.Config,
	txManager repository.TransactionManager,
	rp repository.AdminRepository,
) *AdminService {
	return &AdminService{
		cfg:       cfg,
		txManager: txManager,
		rp:        rp,
	}
}

func (s *AdminService) IsAdmin(ctx context.Context, email string) (bool, error) {
	normalizedEmail, err := normalizeAdminEmail(email)
	if err != nil {
		return false, err
	}

	return s.rp.IsActiveAdminByEmail(ctx, normalizedEmail)
}

func (s *AdminService) CreateSession(
	ctx context.Context,
	input domain.GoogleUserInput,
) (*domain.AdminSession, error) {
	normalized, err := normalizeGoogleUserInput(input)
	if err != nil {
		return nil, err
	}

	isAdmin, err := s.rp.IsActiveAdminByEmail(ctx, normalized.Email)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, ErrInvalidAdminInput
	}

	return s.rp.CreateSession(ctx, domain.CreateAdminSessionInput{
		Email:     normalized.Email,
		FullName:  normalized.FullName,
		GoogleSub: normalized.GoogleSub,
		ExpiresAt: time.Now().UTC().Add(s.cfg.SessionTTL),
	})
}

func (s *AdminService) GetValidSessionByID(ctx context.Context, id uuid.UUID) (*domain.AdminSession, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: id is empty", ErrInvalidAdminInput)
	}

	session, err := s.rp.GetValidSessionByID(ctx, id)
	if err != nil || session == nil {
		return session, err
	}

	isAdmin, err := s.rp.IsActiveAdminByEmail(ctx, session.Email)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, nil
	}

	return session, nil
}

func (s *AdminService) ListAccess(ctx context.Context) ([]domain.AdminAccess, error) {
	return s.rp.ListAccess(ctx)
}

func (s *AdminService) AddAccess(
	ctx context.Context,
	input domain.CreateAdminAccessInput,
) (*domain.AdminAccess, error) {
	normalizedEmail, err := normalizeAdminEmail(input.Email)
	if err != nil {
		return nil, err
	}
	createdBy, err := normalizeAdminEmail(input.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("%w: created_by is empty", ErrInvalidAdminInput)
	}

	return s.rp.UpsertAccess(ctx, domain.CreateAdminAccessInput{
		Email:     normalizedEmail,
		CreatedBy: createdBy,
	})
}

func (s *AdminService) RevokeAccess(ctx context.Context, actorEmail string, email string) error {
	normalizedEmail, err := normalizeAdminEmail(email)
	if err != nil {
		return err
	}
	normalizedActorEmail, err := normalizeAdminEmail(actorEmail)
	if err != nil {
		return fmt.Errorf("%w: actor_email is empty", ErrInvalidAdminInput)
	}
	if strings.EqualFold(normalizedActorEmail, normalizedEmail) {
		return ErrCannotRevokeSelf
	}

	return s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.rp.RevokeAccess(txCtx, normalizedEmail); err != nil {
			return err
		}

		return s.rp.RevokeSessionsByEmail(txCtx, normalizedEmail)
	})
}

func (s *AdminService) ListReports(ctx context.Context) ([]domain.AdminReportRun, error) {
	return s.rp.ListReports(ctx)
}

func (s *AdminService) GetReportRunByPeriod(
	ctx context.Context,
	from time.Time,
	to time.Time,
) (*domain.AdminReportRun, error) {
	from, to, _, err := s.normalizeAdminRange(from, to)
	if err != nil {
		return nil, err
	}
	return s.rp.GetReportRunByPeriod(ctx, from, to)
}

func (s *AdminService) CreateReportRun(
	ctx context.Context,
	input domain.CreateAdminReportRunInput,
) (*domain.AdminReportRun, error) {
	return s.rp.CreateReportRun(ctx, input)
}

func (s *AdminService) ListSessions(ctx context.Context) ([]domain.AdminSession, error) {
	return s.rp.ListSessions(ctx)
}

func (s *AdminService) RevokeSession(ctx context.Context, sessionId uuid.UUID) error {
	if sessionId == uuid.Nil {
		return fmt.Errorf("%w: session_id is empty", ErrInvalidAdminInput)
	}

	return s.rp.RevokeSession(ctx, sessionId)
}

func (s *AdminService) EmployeesMonth(
	ctx context.Context,
	from time.Time,
	to time.Time,
) (*domain.AdminEmployeesMonthOverview, error) {
	from, to, targetMinutes, err := s.normalizeAdminRange(from, to)
	if err != nil {
		return nil, err
	}

	rows, err := s.rp.ListEmployeeMonthRows(ctx, from, to)
	if err != nil {
		return nil, err
	}

	impactedDates, err := s.impactedDates(ctx, from, to)
	if err != nil {
		return nil, err
	}

	employees := s.employeeSummariesFromRows(rows, from, to, targetMinutes, false, impactedDates)
	return &domain.AdminEmployeesMonthOverview{
		From:                from,
		To:                  to,
		WorkdayStart:        s.cfg.WorkdayStart,
		WorkdayEnd:          s.cfg.WorkdayEnd,
		TargetMinutesPerDay: targetMinutes,
		Employees:           employees,
	}, nil
}

func (s *AdminService) EmployeeMonth(
	ctx context.Context,
	userId uuid.UUID,
	from time.Time,
	to time.Time,
) (*domain.AdminEmployeeMonthSummary, error) {
	if userId == uuid.Nil {
		return nil, fmt.Errorf("%w: user_id is empty", ErrInvalidAdminInput)
	}

	from, to, targetMinutes, err := s.normalizeAdminRange(from, to)
	if err != nil {
		return nil, err
	}

	rows, err := s.rp.ListEmployeeMonthRowsByUser(ctx, userId, from, to)
	if err != nil {
		return nil, err
	}

	impactedDates, err := s.impactedDates(ctx, from, to)
	if err != nil {
		return nil, err
	}

	employees := s.employeeSummariesFromRows(rows, from, to, targetMinutes, true, impactedDates)
	if len(employees) == 0 {
		return nil, nil
	}

	return &employees[0], nil
}

func (s *AdminService) SuspiciousActivity(
	ctx context.Context,
	from time.Time,
	to time.Time,
) (*domain.AdminSuspiciousActivity, error) {
	from, to, _, err := s.normalizeAdminRange(from, to)
	if err != nil {
		return nil, err
	}

	events, err := s.rp.ListAttendanceEvents(ctx, from, to)
	if err != nil {
		return nil, err
	}

	return &domain.AdminSuspiciousActivity{
		From:          from,
		To:            to,
		DeviceMatches: suspiciousDevices(events),
		IPMatches:     suspiciousIPs(events, 10*time.Minute),
	}, nil
}

func (s *AdminService) ListSystemOutages(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]domain.SystemOutage, error) {
	from, to, _, err := s.normalizeAdminRange(from, to)
	if err != nil {
		return nil, err
	}

	return s.rp.ListSystemOutages(ctx, from, to)
}

func (s *AdminService) OutageDayEmployees(
	ctx context.Context,
	outageId uuid.UUID,
) (*domain.SystemOutage, []domain.AdminOutageDayEmployeeRow, error) {
	if outageId == uuid.Nil {
		return nil, nil, fmt.Errorf("%w: outage_id is empty", ErrInvalidAdminInput)
	}

	outage, err := s.rp.GetSystemOutageByID(ctx, outageId)
	if err != nil {
		return nil, nil, err
	}
	if outage == nil || outage.AffectedBusinessDate == nil {
		return outage, nil, nil
	}

	rows, err := s.rp.ListOutageDayEmployees(ctx, *outage.AffectedBusinessDate)
	if err != nil {
		return nil, nil, err
	}

	return outage, rows, nil
}

func (s *AdminService) RepairOutageDay(
	ctx context.Context,
	input domain.AdminOutageRepairInput,
) error {
	if input.OutageId == uuid.Nil {
		return fmt.Errorf("%w: outage_id is empty", ErrInvalidAdminInput)
	}
	adminEmail, err := normalizeAdminEmail(input.AdminEmail)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.ResolutionNote) == "" {
		return fmt.Errorf("%w: resolution_note is empty", ErrInvalidAdminInput)
	}
	if len(input.Items) > 0 {
		return fmt.Errorf("%w: system outage does not apply attendance changes; use employee requests", ErrInvalidAdminInput)
	}

	outage, err := s.rp.GetSystemOutageByID(ctx, input.OutageId)
	if err != nil {
		return err
	}
	if outage == nil || outage.AffectedBusinessDate == nil {
		return fmt.Errorf("%w: outage is not repairable", ErrInvalidAdminInput)
	}
	if outage.ResolvedAt != nil && len(input.Items) == 0 {
		return nil
	}

	return s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		return s.rp.ResolveSystemOutage(txCtx, input.OutageId, adminEmail, strings.TrimSpace(input.ResolutionNote))
	})
}

func (s *AdminService) ListExplanations(
	ctx context.Context,
	from time.Time,
	to time.Time,
	status string,
) ([]domain.AdminExplanationRow, error) {
	status = strings.TrimSpace(status)
	if status != "" && status != "pending" && status != "approved" && status != "rejected" && status != "cancelled" {
		return nil, fmt.Errorf("%w: status is invalid", ErrInvalidAdminInput)
	}

	return s.rp.ListExplanations(ctx, from, to, status)
}

func (s *AdminService) VoidAttendanceDay(ctx context.Context, input domain.AdminVoidDayInput) error {
	normalized, err := s.normalizeVoidDayInput(input)
	if err != nil {
		return err
	}

	oldCheckIn, oldCheckOut, err := s.dayEventTimes(ctx, normalized.UserId, normalized.BusinessDate)
	if err != nil {
		return err
	}

	return s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		return s.rp.VoidAttendanceDay(txCtx, normalized, oldCheckIn, oldCheckOut)
	})
}

func (s *AdminService) RestoreAttendanceDay(ctx context.Context, input domain.AdminVoidDayInput) error {
	normalized, err := s.normalizeVoidDayInput(input)
	if err != nil {
		return err
	}

	oldCheckIn, oldCheckOut, err := s.dayEventTimes(ctx, normalized.UserId, normalized.BusinessDate)
	if err != nil {
		return err
	}

	return s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		return s.rp.RestoreAttendanceDay(txCtx, normalized, oldCheckIn, oldCheckOut)
	})
}

func (s *AdminService) ListAuditLogs(ctx context.Context, from time.Time, to time.Time) ([]domain.AdminAuditLog, error) {
	from, to, _, err := s.normalizeAdminRange(from, to)
	if err != nil {
		return nil, err
	}
	return s.rp.ListAuditLogs(ctx, from, to)
}

func (s *AdminService) normalizeVoidDayInput(input domain.AdminVoidDayInput) (domain.AdminVoidDayInput, error) {
	if input.UserId == uuid.Nil {
		return domain.AdminVoidDayInput{}, fmt.Errorf("%w: user_id is empty", ErrInvalidAdminInput)
	}
	adminEmail, err := normalizeAdminEmail(input.AdminEmail)
	if err != nil {
		return domain.AdminVoidDayInput{}, err
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return domain.AdminVoidDayInput{}, fmt.Errorf("%w: reason is empty", ErrInvalidAdminInput)
	}
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return domain.AdminVoidDayInput{}, err
	}
	input.AdminEmail = adminEmail
	input.Reason = reason
	if input.DecisionSource == "" {
		input.DecisionSource = "admin_decision"
	}
	if input.DecisionSource != "admin_decision" && input.DecisionSource != "employee_request" {
		return domain.AdminVoidDayInput{}, fmt.Errorf("%w: decision_source is invalid", ErrInvalidAdminInput)
	}
	input.BusinessDate = normalizeDate(input.BusinessDate, location)
	return input, nil
}

func (s *AdminService) dayEventTimes(ctx context.Context, userId uuid.UUID, businessDate time.Time) (*time.Time, *time.Time, error) {
	rows, err := s.rp.ListEmployeeMonthRowsByUser(ctx, userId, businessDate, businessDate)
	if err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		if row.BusinessDate != nil {
			return row.CheckInAt, row.CheckOutAt, nil
		}
	}
	return nil, nil, nil
}

func (s *AdminService) ApproveExplanation(
	ctx context.Context,
	input domain.AdminExplanationDecisionInput,
) error {
	return s.reviewExplanation(ctx, input, "approved")
}

func (s *AdminService) RejectExplanation(
	ctx context.Context,
	input domain.AdminExplanationDecisionInput,
) error {
	return s.reviewExplanation(ctx, input, "rejected")
}

func (s *AdminService) RollbackExplanationReview(
	ctx context.Context,
	explanationId uuid.UUID,
	adminEmail string,
) error {
	if explanationId == uuid.Nil {
		return fmt.Errorf("%w: explanation_id is empty", ErrInvalidAdminInput)
	}
	normalizedAdminEmail, err := normalizeAdminEmail(adminEmail)
	if err != nil {
		return err
	}

	explanation, err := s.rp.GetExplanationByID(ctx, explanationId)
	if err != nil {
		return err
	}
	if explanation == nil {
		return fmt.Errorf("%w: explanation not found", ErrInvalidAdminInput)
	}
	if explanation.Status == "pending" {
		return fmt.Errorf("%w: explanation is already pending", ErrInvalidAdminInput)
	}

	auditLogs, err := s.rp.ListAuditLogsByExplanation(ctx, explanationId)
	if err != nil {
		return err
	}
	if len(auditLogs) == 0 {
		return fmt.Errorf("%w: explanation has no linked audit trail", ErrInvalidAdminInput)
	}
	lastRollbackIndex := -1
	for index, row := range auditLogs {
		if row.Action == "explanation_rollback" {
			lastRollbackIndex = index
		}
	}
	auditLogs = auditLogs[lastRollbackIndex+1:]
	if len(auditLogs) == 0 {
		return fmt.Errorf("%w: explanation has no active audit trail", ErrInvalidAdminInput)
	}

	return s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		for i := len(auditLogs) - 1; i >= 0; i-- {
			row := auditLogs[i]
			switch row.Action {
			case "check_in_changed":
				if err := s.rp.SetAttendanceEventAt(txCtx, domain.SetAttendanceEventAtInput{
					UserId:       explanation.UserId,
					BusinessDate: explanation.BusinessDate,
					EventType:    AttendanceEventCheckIn,
					EventAt:      row.OldCheckInAt,
					Status:       AttendanceStatusNormal,
				}); err != nil {
					return err
				}
			case "check_out_changed":
				if err := s.rp.SetAttendanceEventAt(txCtx, domain.SetAttendanceEventAtInput{
					UserId:       explanation.UserId,
					BusinessDate: explanation.BusinessDate,
					EventType:    AttendanceEventCheckOut,
					EventAt:      row.OldCheckOutAt,
					Status:       AttendanceStatusNormal,
				}); err != nil {
					return err
				}
			case "day_voided":
				if err := s.rp.RestoreAttendanceDay(txCtx, domain.AdminVoidDayInput{
					UserId:         explanation.UserId,
					BusinessDate:   explanation.BusinessDate,
					AdminEmail:     normalizedAdminEmail,
					Reason:         "Решение по заявке сотрудника отменено",
					DecisionSource: "employee_request",
					ExplanationId:  explanation.Id,
				}, explanation.CheckInAt, explanation.CheckOutAt); err != nil {
					return err
				}
			}
		}

		if _, err := s.rp.ResetExplanationReview(txCtx, explanationId); err != nil {
			return err
		}

		return s.rp.CreateAdminAuditLog(txCtx, domain.AdminAuditInput{
			AdminEmail:     normalizedAdminEmail,
			UserId:         explanation.UserId,
			ExplanationId:  explanation.Id,
			BusinessDate:   explanation.BusinessDate,
			Action:         "explanation_rollback",
			DecisionSource: "employee_request",
			Reason:         "employee_explanation_rollback",
		})
	})
}

func (s *AdminService) reviewExplanation(
	ctx context.Context,
	input domain.AdminExplanationDecisionInput,
	status string,
) error {
	if input.ExplanationId == uuid.Nil {
		return fmt.Errorf("%w: explanation_id is empty", ErrInvalidAdminInput)
	}
	adminEmail, err := normalizeAdminEmail(input.AdminEmail)
	if err != nil {
		return err
	}

	explanation, err := s.rp.GetExplanationByID(ctx, input.ExplanationId)
	if err != nil {
		return err
	}
	if explanation == nil {
		return fmt.Errorf("%w: explanation not found", ErrInvalidAdminInput)
	}
	if input.CheckOutAt != nil && input.CheckInAt == nil && explanation.CheckInAt == nil {
		return fmt.Errorf("%w: check_out requires check_in", ErrInvalidAdminInput)
	}
	if explanation.ReasonType == ExplanationReasonVoidDayRequest && (input.CheckInAt != nil || input.CheckOutAt != nil) {
		return fmt.Errorf("%w: void_day_request cannot include time repair", ErrInvalidAdminInput)
	}
	if explanation.ReasonType == ExplanationReasonLate && input.CheckOutAt != nil {
		return fmt.Errorf("%w: late explanation cannot include check_out repair", ErrInvalidAdminInput)
	}
	if explanation.ReasonType == ExplanationReasonEarlyLeave && input.CheckInAt != nil {
		return fmt.Errorf("%w: early_leave explanation cannot include check_in repair", ErrInvalidAdminInput)
	}
	if explanation.ReasonType == ExplanationReasonMissingCheckIn && input.CheckOutAt != nil {
		return fmt.Errorf("%w: missing_check_in explanation cannot include check_out repair", ErrInvalidAdminInput)
	}
	if explanation.ReasonType == ExplanationReasonMissingCheckOut && input.CheckInAt != nil {
		return fmt.Errorf("%w: missing_check_out explanation cannot include check_in repair", ErrInvalidAdminInput)
	}
	if status == "approved" {
		switch explanation.ReasonType {
		case ExplanationReasonLate:
			if input.CheckInAt == nil {
				return fmt.Errorf("%w: late explanation requires check_in repair", ErrInvalidAdminInput)
			}
			if explanation.CheckInAt != nil && sameRepairClock(*input.CheckInAt, *explanation.CheckInAt) {
				return fmt.Errorf("%w: late explanation requires changed check_in", ErrInvalidAdminInput)
			}
		case ExplanationReasonEarlyLeave:
			if input.CheckOutAt == nil {
				return fmt.Errorf("%w: early_leave explanation requires check_out repair", ErrInvalidAdminInput)
			}
			if explanation.CheckOutAt != nil && sameRepairClock(*input.CheckOutAt, *explanation.CheckOutAt) {
				return fmt.Errorf("%w: early_leave explanation requires changed check_out", ErrInvalidAdminInput)
			}
		case ExplanationReasonMissingDay:
			if input.CheckInAt == nil && explanation.CheckInAt == nil {
				return fmt.Errorf("%w: missing_day requires check_in", ErrInvalidAdminInput)
			}
			if input.CheckOutAt == nil && explanation.CheckOutAt == nil {
				return fmt.Errorf("%w: missing_day requires check_out", ErrInvalidAdminInput)
			}
		case ExplanationReasonMissingCheckIn:
			if input.CheckInAt == nil && explanation.CheckInAt == nil {
				return fmt.Errorf("%w: missing_check_in requires check_in", ErrInvalidAdminInput)
			}
		case ExplanationReasonMissingCheckOut:
			if input.CheckOutAt == nil && explanation.CheckOutAt == nil {
				return fmt.Errorf("%w: missing_check_out requires check_out", ErrInvalidAdminInput)
			}
		}
	}

	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return err
	}
	businessDate := explanation.BusinessDate
	if status == "approved" {
		if err := s.validateExplanationRepairPolicy(explanation.ReasonType, input); err != nil {
			return err
		}
	}
	if input.CheckInAt != nil && isFutureRepairEvent(businessDate, *input.CheckInAt, location) {
		return fmt.Errorf("%w: check_in cannot be in the future", ErrInvalidAdminInput)
	}
	if input.CheckOutAt != nil && isFutureRepairEvent(businessDate, *input.CheckOutAt, location) {
		return fmt.Errorf("%w: check_out cannot be in the future", ErrInvalidAdminInput)
	}
	note := strings.TrimSpace(input.ReviewNote)

	return s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if status == "approved" {
			if explanation.ReasonType == ExplanationReasonVoidDayRequest {
				reason := strings.TrimSpace(note)
				if reason == "" {
					reason = strings.TrimSpace(explanation.Comment)
				}
				if err := s.rp.VoidAttendanceDay(txCtx, domain.AdminVoidDayInput{
					UserId:         explanation.UserId,
					BusinessDate:   businessDate,
					AdminEmail:     adminEmail,
					Reason:         reason,
					DecisionSource: "employee_request",
					ExplanationId:  explanation.Id,
				}, explanation.CheckInAt, explanation.CheckOutAt); err != nil {
					return err
				}

				if _, err := s.rp.UpdateExplanationReview(txCtx, input.ExplanationId, status, adminEmail, note); err != nil {
					return err
				}
				return s.logExplanationReviewAudit(txCtx, explanation, status, adminEmail, note)
			}
			if input.CheckInAt != nil {
				eventAt := repairEventTime(businessDate, *input.CheckInAt, location)
				if err := s.applyExplanationRepairEvent(
					txCtx,
					explanation.UserId,
					businessDate,
					AttendanceEventCheckIn,
					eventAt,
					adminEmail,
					explanation.Id,
				); err != nil {
					return err
				}
			}
			if input.CheckOutAt != nil {
				eventAt := repairEventTime(businessDate, *input.CheckOutAt, location)
				if err := s.applyExplanationRepairEvent(
					txCtx,
					explanation.UserId,
					businessDate,
					AttendanceEventCheckOut,
					eventAt,
					adminEmail,
					explanation.Id,
				); err != nil {
					return err
				}
			}
		}

		if _, err := s.rp.UpdateExplanationReview(txCtx, input.ExplanationId, status, adminEmail, note); err != nil {
			return err
		}
		return s.logExplanationReviewAudit(txCtx, explanation, status, adminEmail, note)
	})
}

func (s *AdminService) validateExplanationRepairPolicy(
	reasonType string,
	input domain.AdminExplanationDecisionInput,
) error {
	switch reasonType {
	case ExplanationReasonLate:
		if input.CheckInAt == nil {
			return nil
		}
		startClock, err := parseWorkdayClock(s.cfg.WorkdayStart)
		if err != nil {
			return err
		}
		if clockBefore(*input.CheckInAt, startClock) {
			return fmt.Errorf("%w: check_in cannot be before workday start for late explanation", ErrInvalidAdminInput)
		}
	case ExplanationReasonEarlyLeave:
		if input.CheckOutAt == nil {
			return nil
		}
		endClock, err := parseWorkdayClock(s.cfg.WorkdayEnd)
		if err != nil {
			return err
		}
		if clockAfter(*input.CheckOutAt, endClock) {
			return fmt.Errorf("%w: check_out cannot be after workday end for early leave explanation", ErrInvalidAdminInput)
		}
	}
	return nil
}

func clockBefore(value time.Time, clock workdayClock) bool {
	return value.Hour()*60+value.Minute() < clock.Hour*60+clock.Minute
}

func clockAfter(value time.Time, clock workdayClock) bool {
	return value.Hour()*60+value.Minute() > clock.Hour*60+clock.Minute
}

func sameRepairClock(left time.Time, right time.Time) bool {
	return left.Hour() == right.Hour() && left.Minute() == right.Minute()
}

func (s *AdminService) logExplanationReviewAudit(
	ctx context.Context,
	explanation *domain.AdminExplanationRow,
	status string,
	adminEmail string,
	note string,
) error {
	action := "explanation_approved"
	reason := "employee_explanation_approved"
	if status == "rejected" {
		action = "explanation_rejected"
		reason = "employee_explanation_rejected"
	}
	if strings.TrimSpace(note) != "" {
		reason = strings.TrimSpace(note)
	}

	return s.rp.CreateAdminAuditLog(ctx, domain.AdminAuditInput{
		AdminEmail:     adminEmail,
		UserId:         explanation.UserId,
		ExplanationId:  explanation.Id,
		BusinessDate:   explanation.BusinessDate,
		Action:         action,
		DecisionSource: "employee_request",
		Reason:         reason,
	})
}

func (s *AdminService) applyExplanationRepairEvent(
	ctx context.Context,
	userId uuid.UUID,
	businessDate time.Time,
	eventType string,
	eventAt time.Time,
	adminEmail string,
	explanationId uuid.UUID,
) error {
	result, err := s.rp.UpsertAttendanceEventAt(ctx, domain.UpsertAttendanceEventAtInput{
		UserId:       userId,
		BusinessDate: businessDate,
		EventType:    eventType,
		EventAt:      eventAt,
		Status:       AttendanceStatusNormal,
	})
	if err != nil {
		return err
	}
	if result.OldEventAt != nil && result.OldEventAt.Equal(result.NewEventAt) {
		return nil
	}

	return s.rp.CreateAttendanceAdjustment(ctx, domain.CreateAttendanceAdjustmentInput{
		UserId:              userId,
		BusinessDate:        businessDate,
		EventType:           eventType,
		OldEventAt:          result.OldEventAt,
		NewEventAt:          result.NewEventAt,
		Reason:              "employee_explanation_approved",
		CreatedByAdminEmail: adminEmail,
		DecisionSource:      "employee_request",
		ExplanationId:       explanationId,
	})
}

func (s *AdminService) applyOutageRepairEvent(
	ctx context.Context,
	userId uuid.UUID,
	businessDate time.Time,
	eventType string,
	eventAt time.Time,
	outageId uuid.UUID,
	adminEmail string,
) error {
	result, err := s.rp.UpsertAttendanceEventAt(ctx, domain.UpsertAttendanceEventAtInput{
		UserId:       userId,
		BusinessDate: businessDate,
		EventType:    eventType,
		EventAt:      eventAt,
		Status:       AttendanceStatusSystemOutage,
	})
	if err != nil {
		return err
	}
	if result.OldEventAt != nil && result.OldEventAt.Equal(result.NewEventAt) {
		return nil
	}

	return s.rp.CreateAttendanceAdjustment(ctx, domain.CreateAttendanceAdjustmentInput{
		UserId:              userId,
		BusinessDate:        businessDate,
		EventType:           eventType,
		OldEventAt:          result.OldEventAt,
		NewEventAt:          result.NewEventAt,
		Reason:              "system_outage_repair",
		OutageId:            outageId,
		CreatedByAdminEmail: adminEmail,
		DecisionSource:      "admin_decision",
	})
}

func repairEventTime(businessDate time.Time, clock time.Time, location *time.Location) time.Time {
	date := businessDate.In(location)
	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		clock.Hour(),
		clock.Minute(),
		0,
		0,
		location,
	)
}

func isFutureRepairEvent(businessDate time.Time, clock time.Time, location *time.Location) bool {
	eventAt := repairEventTime(businessDate, clock, location)
	now := time.Now().In(location)
	return sameBusinessDate(businessDate, now, location) && eventAt.After(now)
}

func normalizeAdminEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", fmt.Errorf("%w: email is empty", ErrInvalidAdminInput)
	}
	if !strings.Contains(normalized, "@") {
		return "", fmt.Errorf("%w: email is invalid", ErrInvalidAdminInput)
	}

	return normalized, nil
}

func (s *AdminService) normalizeAdminRange(
	from time.Time,
	to time.Time,
) (time.Time, time.Time, int, error) {
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}

	from = normalizeDate(from, location)
	to = normalizeDate(to, location)
	if to.Before(from) {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("%w: to is before from", ErrInvalidAdminInput)
	}
	if to.Sub(from) > 370*24*time.Hour {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("%w: range is too large", ErrInvalidAdminInput)
	}

	startClock, err := parseWorkdayClock(s.cfg.WorkdayStart)
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	endClock, err := parseWorkdayClock(s.cfg.WorkdayEnd)
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}

	return from, to, workdayTargetMinutes(startClock, endClock), nil
}

func (s *AdminService) employeeSummariesFromRows(
	rows []domain.AdminEmployeeMonthRow,
	from time.Time,
	to time.Time,
	targetMinutesPerDay int,
	includeDays bool,
	impactedDates map[string]bool,
) []domain.AdminEmployeeMonthSummary {
	byUser := make(map[uuid.UUID]*domain.AdminEmployeeMonthSummary)
	order := make([]uuid.UUID, 0)

	for _, row := range rows {
		summary, ok := byUser[row.UserId]
		if !ok {
			summary = &domain.AdminEmployeeMonthSummary{
				UserId:   row.UserId,
				Email:    row.Email,
				FullName: row.FullName,
			}
			byUser[row.UserId] = summary
			order = append(order, row.UserId)
		}

		if row.BusinessDate == nil {
			continue
		}

		day := s.attendanceDaySummary(*row.BusinessDate, row.CheckInAt, row.CheckOutAt)
		day.ImpactedByOutage = impactedDates[day.Date.Format("2006-01-02")]
		if row.Voided {
			day.Voided = true
			day.VoidReason = row.VoidReason
			day.VoidedByAdmin = row.VoidedByAdmin
			day.VoidedAt = row.VoidedAt
			day.WorkedMinutes = 0
			day.LateMinutes = 0
			day.EarlyLeaveMinutes = 0
		}
		summary.AttendanceDays = append(summary.AttendanceDays, day)
		if row.Voided {
			continue
		}
		summary.WorkedMinutes += day.WorkedMinutes
		if row.CheckInAt != nil {
			summary.CheckInCount++
		}
		if row.CheckOutAt != nil {
			summary.CheckOutCount++
		}
		if day.LateMinutes > 0 {
			summary.LateCount++
		}
		if day.EarlyLeaveMinutes > 0 {
			summary.EarlyLeaveCount++
		}
		if row.CheckInAt != nil && row.CheckOutAt == nil {
			summary.MissingCheckOuts++
		}
		if day.Status != AttendanceSummaryStatusEmpty {
			summary.WorkedDays++
		}
	}

	for _, summary := range byUser {
		summary.TargetMinutes = summary.CheckInCount * targetMinutesPerDay
		if includeDays {
			summary.AttendanceDays = fillAdminMonthDays(
				from,
				to,
				summary.AttendanceDays,
				impactedDates,
			)
			continue
		}
		summary.AttendanceDays = nil
	}

	result := make([]domain.AdminEmployeeMonthSummary, 0, len(order))
	for _, userId := range order {
		result = append(result, *byUser[userId])
	}

	sort.SliceStable(result, func(i int, j int) bool {
		if result[i].LateCount != result[j].LateCount {
			return result[i].LateCount > result[j].LateCount
		}
		if result[i].EarlyLeaveCount != result[j].EarlyLeaveCount {
			return result[i].EarlyLeaveCount > result[j].EarlyLeaveCount
		}
		return result[i].FullName < result[j].FullName
	})

	return result
}

func (s *AdminService) attendanceDaySummary(
	businessDate time.Time,
	checkInAt *time.Time,
	checkOutAt *time.Time,
) domain.AttendanceDaySummary {
	day := domain.AttendanceDaySummary{
		Date:       businessDate,
		CheckInAt:  checkInAt,
		CheckOutAt: checkOutAt,
		Status:     AttendanceSummaryStatusEmpty,
	}

	if checkInAt != nil {
		day.LateMinutes = s.adminLateMinutes(businessDate, *checkInAt)
		day.Status = AttendanceSummaryStatusInProgress
	}
	if checkOutAt != nil {
		day.EarlyLeaveMinutes = s.adminEarlyLeaveMinutes(businessDate, *checkOutAt)
	}
	if checkInAt != nil && checkOutAt != nil {
		day.WorkedMinutes = positiveMinutes(checkOutAt.Sub(*checkInAt))
		day.Status = AttendanceSummaryStatusComplete
	}

	return day
}

func (s *AdminService) adminLateMinutes(businessDate time.Time, checkInAt time.Time) int {
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return 0
	}
	if isWeekend(businessDate, location) {
		return 0
	}
	startClock, err := parseWorkdayClock(s.cfg.WorkdayStart)
	if err != nil {
		return 0
	}
	localDate := businessDate.In(location)

	startAt := time.Date(
		localDate.Year(), localDate.Month(), localDate.Day(),
		startClock.Hour, startClock.Minute, 0, 0, location,
	)
	return positiveMinutes(checkInAt.In(location).Sub(startAt))
}

func (s *AdminService) adminEarlyLeaveMinutes(businessDate time.Time, checkOutAt time.Time) int {
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return 0
	}
	if isWeekend(businessDate, location) {
		return 0
	}
	endClock, err := parseWorkdayClock(s.cfg.WorkdayEnd)
	if err != nil {
		return 0
	}
	localDate := businessDate.In(location)

	endAt := time.Date(
		localDate.Year(), localDate.Month(), localDate.Day(),
		endClock.Hour, endClock.Minute, 0, 0, location,
	)
	return positiveMinutes(endAt.Sub(checkOutAt.In(location)))
}

func fillAdminMonthDays(
	from time.Time,
	to time.Time,
	days []domain.AttendanceDaySummary,
	impactedDates map[string]bool,
) []domain.AttendanceDaySummary {
	byDate := make(map[string]domain.AttendanceDaySummary, len(days))
	for _, day := range days {
		byDate[day.Date.Format("2006-01-02")] = day
	}

	result := make([]domain.AttendanceDaySummary, 0, int(to.Sub(from).Hours()/24)+1)
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		dateKey := date.Format("2006-01-02")
		day, ok := byDate[dateKey]
		if !ok {
			day = domain.AttendanceDaySummary{
				Date:   date,
				Status: AttendanceSummaryStatusEmpty,
			}
		}
		day.ImpactedByOutage = impactedDates[dateKey]
		result = append(result, day)
	}

	return result
}

func (s *AdminService) impactedDates(ctx context.Context, from time.Time, to time.Time) (map[string]bool, error) {
	outages, err := s.rp.ListSystemOutages(ctx, from, to)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool, len(outages))
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return nil, err
	}
	for _, outage := range outages {
		addImpactedOutageDates(
			result,
			outage,
			from,
			to,
			location,
			s.cfg.OutageImpactStart,
			s.cfg.OutageImpactEnd,
		)
	}
	return result, nil
}

func suspiciousDevices(events []domain.AdminAttendanceEventRow) []domain.AdminSuspiciousDeviceMatch {
	byDevice := make(map[string][]domain.AdminAttendanceEventRow)
	for _, event := range events {
		if event.DeviceId == "" || event.DeviceId == "unknown" {
			continue
		}
		byDevice[event.DeviceId] = append(byDevice[event.DeviceId], event)
	}

	result := make([]domain.AdminSuspiciousDeviceMatch, 0)
	for deviceId, deviceEvents := range byDevice {
		sort.Slice(deviceEvents, func(i int, j int) bool {
			return deviceEvents[i].EventAt.Before(deviceEvents[j].EventAt)
		})
		if len(deviceEvents) == 0 {
			continue
		}

		owner := deviceEvents[0]
		for _, event := range deviceEvents[1:] {
			if event.UserId == owner.UserId {
				continue
			}

			result = append(result, domain.AdminSuspiciousDeviceMatch{
				DeviceId: deviceId,
				Owner: domain.AdminSuspiciousActor{
					UserId:   owner.UserId,
					Email:    owner.Email,
					FullName: owner.FullName,
				},
				Event: event,
			})
		}
	}

	sort.Slice(result, func(i int, j int) bool {
		return result[i].Event.EventAt.After(result[j].Event.EventAt)
	})

	return result
}

func suspiciousIPs(
	events []domain.AdminAttendanceEventRow,
	window time.Duration,
) []domain.AdminSuspiciousIPMatch {
	byIP := make(map[string][]domain.AdminAttendanceEventRow)
	for _, event := range events {
		if event.ExternalIp == "" || event.ExternalIp == "unknown" {
			continue
		}
		byIP[event.ExternalIp] = append(byIP[event.ExternalIp], event)
	}

	result := make([]domain.AdminSuspiciousIPMatch, 0)
	for ip, ipEvents := range byIP {
		sort.Slice(ipEvents, func(i int, j int) bool {
			return ipEvents[i].EventAt.Before(ipEvents[j].EventAt)
		})

		for j := 0; j < len(ipEvents); j++ {
			var previous *domain.AdminAttendanceEventRow
			for i := j - 1; i >= 0; i-- {
				diff := ipEvents[j].EventAt.Sub(ipEvents[i].EventAt)
				if diff > window {
					break
				}
				if ipEvents[i].UserId == ipEvents[j].UserId {
					continue
				}
				previous = &ipEvents[i]
				break
			}
			if previous == nil {
				continue
			}

			result = append(result, domain.AdminSuspiciousIPMatch{
				ExternalIp:     ip,
				Event:          ipEvents[j],
				PreviousEvent:  *previous,
				MinutesBetween: int(ipEvents[j].EventAt.Sub(previous.EventAt).Minutes()),
			})
		}
	}

	sort.Slice(result, func(i int, j int) bool {
		return result[i].Event.EventAt.After(result[j].Event.EventAt)
	})

	return result
}
