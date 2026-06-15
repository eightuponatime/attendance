package impl

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"attendance/config"
	"attendance/internal/domain"
	"attendance/internal/repository"

	"github.com/google/uuid"
)

const (
	AttendanceEventCheckIn  = "check_in"
	AttendanceEventCheckOut = "check_out"

	AttendanceStatusNormal       = "normal"
	AttendanceStatusSystemOutage = "system_outage"

	AttendanceSummaryStatusEmpty      = "empty"
	AttendanceSummaryStatusInProgress = "in_progress"
	AttendanceSummaryStatusComplete   = "complete"

	ExplanationReasonLate            = "late"
	ExplanationReasonEarlyLeave      = "early_leave"
	ExplanationReasonMissingCheckIn  = "missing_check_in"
	ExplanationReasonMissingCheckOut = "missing_check_out"
	ExplanationReasonMissingDay      = "missing_day"
)

var (
	ErrInvalidAttendanceInput = errors.New("invalid attendance input")
	ErrAttendanceAlreadyDone  = errors.New("attendance event already exists")
	ErrCheckInRequired        = errors.New("check-in is required before check-out")
	ErrInvalidAttendanceRange = errors.New("invalid attendance range")
	ErrExplanationUnavailable = errors.New("explanation is not available for this day")
)

type AttendanceService struct {
	cfg       *config.Config
	txManager repository.TransactionManager
	rp        repository.AttendanceRepository
	systemRp  repository.SystemRepository
}

func NewAttendanceService(
	cfg *config.Config,
	txManager repository.TransactionManager,
	rp repository.AttendanceRepository,
	systemRp repository.SystemRepository,
) *AttendanceService {
	return &AttendanceService{
		cfg:       cfg,
		txManager: txManager,
		rp:        rp,
		systemRp:  systemRp,
	}
}

func (s *AttendanceService) Today(ctx context.Context, userId uuid.UUID) (*domain.AttendanceToday, error) {
	if userId == uuid.Nil {
		return nil, fmt.Errorf("%w: user_id is empty", ErrInvalidAttendanceInput)
	}

	businessDate, err := s.businessDate()
	if err != nil {
		return nil, err
	}

	record, err := s.rp.GetRecordByUserAndDate(ctx, userId, businessDate)
	if err != nil {
		return nil, err
	}

	today, err := s.todayFromRecord(ctx, businessDate, record)
	if err != nil {
		return nil, err
	}
	today.ImpactedByOutage = s.isImpactedByOutage(ctx, businessDate)
	return today, nil
}

func (s *AttendanceService) CheckIn(
	ctx context.Context,
	input domain.AttendanceMarkInput,
) (*domain.AttendanceToday, error) {
	normalized, err := s.normalizeMarkInput(input)
	if err != nil {
		return nil, err
	}

	businessDate, err := s.businessDate()
	if err != nil {
		return nil, err
	}

	var today *domain.AttendanceToday
	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		record, err := s.rp.CreateOrGetRecord(txCtx, normalized.UserId, businessDate)
		if err != nil {
			return err
		}

		checkIn, err := s.rp.GetEventByRecordAndType(txCtx, record.Id, AttendanceEventCheckIn)
		if err != nil {
			return err
		}
		if checkIn != nil {
			return ErrAttendanceAlreadyDone
		}

		checkIn, err = s.rp.CreateEvent(txCtx, domain.CreateAttendanceEventInput{
			RecordId:   record.Id,
			EventType:  AttendanceEventCheckIn,
			Status:     AttendanceStatusNormal,
			PhoneModel: normalized.PhoneModel,
			Browser:    normalized.Browser,
			DeviceId:   normalized.DeviceId,
			ExternalIp: normalized.ExternalIp,
		})
		if err != nil {
			return err
		}

		today = &domain.AttendanceToday{
			BusinessDate: businessDate,
			Record:       record,
			CheckIn:      checkIn,
			LateMinutes:  s.lateMinutes(businessDate, checkIn.EventAt),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	today.ImpactedByOutage = s.isImpactedByOutage(ctx, businessDate)

	return today, nil
}

func (s *AttendanceService) CheckOut(
	ctx context.Context,
	input domain.AttendanceMarkInput,
) (*domain.AttendanceToday, error) {
	normalized, err := s.normalizeMarkInput(input)
	if err != nil {
		return nil, err
	}

	businessDate, err := s.businessDate()
	if err != nil {
		return nil, err
	}

	var today *domain.AttendanceToday
	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		record, err := s.rp.GetRecordByUserAndDate(txCtx, normalized.UserId, businessDate)
		if err != nil {
			return err
		}
		if record == nil {
			return ErrCheckInRequired
		}

		checkIn, err := s.rp.GetEventByRecordAndType(txCtx, record.Id, AttendanceEventCheckIn)
		if err != nil {
			return err
		}
		if checkIn == nil {
			return ErrCheckInRequired
		}

		checkOut, err := s.rp.GetEventByRecordAndType(txCtx, record.Id, AttendanceEventCheckOut)
		if err != nil {
			return err
		}
		if checkOut != nil {
			return ErrAttendanceAlreadyDone
		}

		checkOut, err = s.rp.CreateEvent(txCtx, domain.CreateAttendanceEventInput{
			RecordId:   record.Id,
			EventType:  AttendanceEventCheckOut,
			Status:     AttendanceStatusNormal,
			PhoneModel: normalized.PhoneModel,
			Browser:    normalized.Browser,
			DeviceId:   normalized.DeviceId,
			ExternalIp: normalized.ExternalIp,
		})
		if err != nil {
			return err
		}

		today = &domain.AttendanceToday{
			BusinessDate:      businessDate,
			Record:            record,
			CheckIn:           checkIn,
			CheckOut:          checkOut,
			LateMinutes:       s.lateMinutes(businessDate, checkIn.EventAt),
			EarlyLeaveMinutes: s.earlyLeaveMinutes(businessDate, checkOut.EventAt),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	today.ImpactedByOutage = s.isImpactedByOutage(ctx, businessDate)

	return today, nil
}

func (s *AttendanceService) Summary(
	ctx context.Context,
	userId uuid.UUID,
	from time.Time,
	to time.Time,
) (*domain.AttendanceSummary, error) {
	if userId == uuid.Nil {
		return nil, fmt.Errorf("%w: user_id is empty", ErrInvalidAttendanceInput)
	}

	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return nil, err
	}

	from = normalizeDate(from, location)
	to = normalizeDate(to, location)
	if to.Before(from) {
		return nil, fmt.Errorf("%w: to is before from", ErrInvalidAttendanceRange)
	}
	if to.Sub(from) > 370*24*time.Hour {
		return nil, fmt.Errorf("%w: range is too large", ErrInvalidAttendanceRange)
	}

	startClock, err := parseWorkdayClock(s.cfg.WorkdayStart)
	if err != nil {
		return nil, err
	}
	endClock, err := parseWorkdayClock(s.cfg.WorkdayEnd)
	if err != nil {
		return nil, err
	}

	targetMinutes := workdayTargetMinutes(startClock, endClock)
	rows, err := s.rp.GetRangeEventRows(ctx, userId, from, to)
	if err != nil {
		return nil, err
	}

	rowsByDate := make(map[string]domain.AttendanceRangeEventRow, len(rows))
	for _, row := range rows {
		rowsByDate[row.BusinessDate.In(location).Format("2006-01-02")] = row
	}
	impactedDates := s.impactedDates(ctx, from, to)
	explanations, err := s.rp.ListExplanationsByUserRange(ctx, userId, from, to)
	if err != nil {
		return nil, err
	}
	explanationsByDate := make(map[string][]domain.AttendanceExplanation)
	for _, explanation := range explanations {
		key := explanation.BusinessDate.In(location).Format("2006-01-02")
		explanationsByDate[key] = append(explanationsByDate[key], explanation)
	}

	days := make([]domain.AttendanceDaySummary, 0, int(to.Sub(from).Hours()/24)+1)
	now := time.Now()
	for date := from; !date.After(to); date = date.AddDate(0, 0, 1) {
		dateKey := date.Format("2006-01-02")
		row, ok := rowsByDate[date.Format("2006-01-02")]
		if !ok {
			days = append(days, domain.AttendanceDaySummary{
				Date:             date,
				Status:           AttendanceSummaryStatusEmpty,
				ImpactedByOutage: impactedDates[dateKey],
				Explanations:     explanationsByDate[dateKey],
			})
			continue
		}

		day := domain.AttendanceDaySummary{
			Date:             date,
			CheckInAt:        row.CheckInAt,
			CheckOutAt:       row.CheckOutAt,
			Status:           AttendanceSummaryStatusComplete,
			ImpactedByOutage: impactedDates[dateKey],
			Explanations:     explanationsByDate[dateKey],
		}

		if row.CheckInAt == nil {
			day.Status = AttendanceSummaryStatusEmpty
		} else {
			workdayStart := workdayTime(date, location, startClock)
			if row.CheckInAt.After(workdayStart) {
				day.LateMinutes = int(math.Ceil(row.CheckInAt.Sub(workdayStart).Minutes()))
			}

			if row.CheckOutAt == nil {
				day.Status = AttendanceSummaryStatusInProgress
				if sameBusinessDate(date, now, location) {
					day.WorkedMinutes = positiveMinutes(now.Sub(*row.CheckInAt))
				}
			} else {
				day.WorkedMinutes = positiveMinutes(row.CheckOutAt.Sub(*row.CheckInAt))
				workdayEnd := workdayTime(date, location, endClock)
				if row.CheckOutAt.Before(workdayEnd) {
					day.EarlyLeaveMinutes = int(math.Ceil(workdayEnd.Sub(*row.CheckOutAt).Minutes()))
				}
			}
		}

		days = append(days, day)
	}

	return &domain.AttendanceSummary{
		From:                from,
		To:                  to,
		WorkdayStart:        s.cfg.WorkdayStart,
		WorkdayEnd:          s.cfg.WorkdayEnd,
		TargetMinutesPerDay: targetMinutes,
		Days:                days,
	}, nil
}

func (s *AttendanceService) SubmitExplanation(
	ctx context.Context,
	input domain.CreateAttendanceExplanationInput,
) (*domain.AttendanceExplanation, error) {
	normalized, err := s.normalizeExplanationInput(input)
	if err != nil {
		return nil, err
	}

	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return nil, err
	}
	businessDate := normalizeDate(normalized.BusinessDate, location)

	record, err := s.rp.GetRecordByUserAndDate(ctx, normalized.UserId, businessDate)
	if err != nil {
		return nil, err
	}
	today, err := s.todayFromRecord(ctx, businessDate, record)
	if err != nil {
		return nil, err
	}

	if !s.explanationAllowed(businessDate, today, normalized.ReasonType, location) {
		return nil, ErrExplanationUnavailable
	}

	normalized.BusinessDate = businessDate
	return s.rp.UpsertExplanation(ctx, normalized)
}

func (s *AttendanceService) normalizeExplanationInput(
	input domain.CreateAttendanceExplanationInput,
) (domain.CreateAttendanceExplanationInput, error) {
	normalized := domain.CreateAttendanceExplanationInput{
		UserId:       input.UserId,
		BusinessDate: input.BusinessDate,
		ReasonType:   strings.TrimSpace(input.ReasonType),
		Comment:      strings.Join(strings.Fields(strings.TrimSpace(input.Comment)), " "),
	}

	if normalized.UserId == uuid.Nil {
		return domain.CreateAttendanceExplanationInput{}, fmt.Errorf("%w: user_id is empty", ErrInvalidAttendanceInput)
	}
	if normalized.BusinessDate.IsZero() {
		return domain.CreateAttendanceExplanationInput{}, fmt.Errorf("%w: business_date is empty", ErrInvalidAttendanceInput)
	}
	if normalized.Comment == "" {
		return domain.CreateAttendanceExplanationInput{}, fmt.Errorf("%w: comment is empty", ErrInvalidAttendanceInput)
	}
	if len([]rune(normalized.Comment)) > 1000 {
		return domain.CreateAttendanceExplanationInput{}, fmt.Errorf("%w: comment is too long", ErrInvalidAttendanceInput)
	}

	switch normalized.ReasonType {
	case ExplanationReasonLate,
		ExplanationReasonEarlyLeave,
		ExplanationReasonMissingCheckIn,
		ExplanationReasonMissingCheckOut,
		ExplanationReasonMissingDay:
	default:
		return domain.CreateAttendanceExplanationInput{}, fmt.Errorf("%w: reason_type is invalid", ErrInvalidAttendanceInput)
	}

	return normalized, nil
}

func (s *AttendanceService) explanationAllowed(
	businessDate time.Time,
	today *domain.AttendanceToday,
	reasonType string,
	location *time.Location,
) bool {
	now := time.Now().In(location)
	afterReviewTime := !sameBusinessDate(businessDate, now, location) ||
		!now.Before(workdayTime(businessDate, location, workdayClock{Hour: 18}))

	switch reasonType {
	case ExplanationReasonLate:
		return today.CheckIn != nil && today.LateMinutes > 0
	case ExplanationReasonEarlyLeave:
		return today.CheckOut != nil && today.EarlyLeaveMinutes > 0
	case ExplanationReasonMissingCheckOut:
		return afterReviewTime && today.CheckIn != nil && today.CheckOut == nil
	case ExplanationReasonMissingCheckIn:
		return afterReviewTime && today.CheckIn == nil && today.CheckOut != nil
	case ExplanationReasonMissingDay:
		return afterReviewTime && !isWeekend(businessDate, location) && today.CheckIn == nil && today.CheckOut == nil
	default:
		return false
	}
}

func isWeekend(date time.Time, location *time.Location) bool {
	weekday := date.In(location).Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

func (s *AttendanceService) todayFromRecord(
	ctx context.Context,
	businessDate time.Time,
	record *domain.AttendanceRecords,
) (*domain.AttendanceToday, error) {
	today := &domain.AttendanceToday{
		BusinessDate: businessDate,
		Record:       record,
	}
	if record == nil {
		return today, nil
	}

	checkIn, err := s.rp.GetEventByRecordAndType(ctx, record.Id, AttendanceEventCheckIn)
	if err != nil {
		return nil, err
	}

	checkOut, err := s.rp.GetEventByRecordAndType(ctx, record.Id, AttendanceEventCheckOut)
	if err != nil {
		return nil, err
	}

	today.CheckIn = checkIn
	today.CheckOut = checkOut
	if checkIn != nil {
		today.LateMinutes = s.lateMinutes(businessDate, checkIn.EventAt)
	}
	if checkOut != nil {
		today.EarlyLeaveMinutes = s.earlyLeaveMinutes(businessDate, checkOut.EventAt)
	}
	return today, nil
}

func (s *AttendanceService) normalizeMarkInput(input domain.AttendanceMarkInput) (domain.AttendanceMarkInput, error) {
	normalized := domain.AttendanceMarkInput{
		UserId:     input.UserId,
		PhoneModel: strings.TrimSpace(input.PhoneModel),
		Browser:    strings.TrimSpace(input.Browser),
		DeviceId:   strings.TrimSpace(input.DeviceId),
		ExternalIp: strings.TrimSpace(input.ExternalIp),
	}

	if normalized.UserId == uuid.Nil {
		return domain.AttendanceMarkInput{}, fmt.Errorf("%w: user_id is empty", ErrInvalidAttendanceInput)
	}
	if normalized.DeviceId == "" {
		return domain.AttendanceMarkInput{}, fmt.Errorf("%w: device_id is empty", ErrInvalidAttendanceInput)
	}
	if normalized.PhoneModel == "" {
		normalized.PhoneModel = "unknown"
	}
	if normalized.Browser == "" {
		normalized.Browser = "unknown"
	}
	if normalized.ExternalIp == "" {
		normalized.ExternalIp = "unknown"
	}

	return normalized, nil
}

func (s *AttendanceService) businessDate() (time.Time, error) {
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now().In(location)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location), nil
}

type workdayClock struct {
	Hour   int
	Minute int
}

func parseWorkdayClock(value string) (workdayClock, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return workdayClock{}, err
	}

	return workdayClock{
		Hour:   parsed.Hour(),
		Minute: parsed.Minute(),
	}, nil
}

func workdayTime(date time.Time, location *time.Location, clock workdayClock) time.Time {
	localDate := date.In(location)
	return time.Date(
		localDate.Year(),
		localDate.Month(),
		localDate.Day(),
		clock.Hour,
		clock.Minute,
		0,
		0,
		location,
	)
}

func workdayTargetMinutes(start workdayClock, end workdayClock) int {
	startMinutes := start.Hour*60 + start.Minute
	endMinutes := end.Hour*60 + end.Minute
	if endMinutes <= startMinutes {
		endMinutes += 24 * 60
	}

	return endMinutes - startMinutes
}

func (s *AttendanceService) lateMinutes(businessDate time.Time, checkInAt time.Time) int {
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return 0
	}

	startClock, err := parseWorkdayClock(s.cfg.WorkdayStart)
	if err != nil {
		return 0
	}

	workdayStart := workdayTime(businessDate, location, startClock)
	if !checkInAt.After(workdayStart) {
		return 0
	}

	return int(math.Ceil(checkInAt.Sub(workdayStart).Minutes()))
}

func (s *AttendanceService) earlyLeaveMinutes(businessDate time.Time, checkOutAt time.Time) int {
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return 0
	}

	endClock, err := parseWorkdayClock(s.cfg.WorkdayEnd)
	if err != nil {
		return 0
	}

	workdayEnd := workdayTime(businessDate, location, endClock)
	if !checkOutAt.Before(workdayEnd) {
		return 0
	}

	return int(math.Ceil(workdayEnd.Sub(checkOutAt).Minutes()))
}

func normalizeDate(date time.Time, location *time.Location) time.Time {
	localDate := date.In(location)
	return time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, location)
}

func (s *AttendanceService) isImpactedByOutage(ctx context.Context, businessDate time.Time) bool {
	return s.impactedDates(ctx, businessDate, businessDate)[businessDate.Format("2006-01-02")]
}

func (s *AttendanceService) impactedDates(ctx context.Context, from time.Time, to time.Time) map[string]bool {
	result := make(map[string]bool)
	if s.systemRp == nil {
		return result
	}

	outages, err := s.systemRp.ListSystemOutages(ctx, from, to)
	if err != nil {
		return result
	}
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return result
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
	return result
}

func addImpactedOutageDates(
	result map[string]bool,
	outage domain.SystemOutage,
	from time.Time,
	to time.Time,
	location *time.Location,
	impactStart string,
	impactEnd string,
) {
	if !outage.ImpactsWorkHours {
		return
	}

	startClock, err := time.Parse("15:04", impactStart)
	if err != nil {
		return
	}
	endClock, err := time.Parse("15:04", impactEnd)
	if err != nil {
		return
	}

	fromDate := normalizeDate(from, location)
	toDate := normalizeDate(to, location)
	for day := normalizeDate(outage.StartedAt.In(location), location); !day.After(normalizeDate(outage.EndedAt.In(location), location)); day = day.AddDate(0, 0, 1) {
		if !isBusinessWeekday(day) {
			continue
		}
		if day.Before(fromDate) || day.After(toDate) {
			continue
		}

		windowStart := time.Date(day.Year(), day.Month(), day.Day(), startClock.Hour(), startClock.Minute(), 0, 0, location)
		windowEnd := time.Date(day.Year(), day.Month(), day.Day(), endClock.Hour(), endClock.Minute(), 0, 0, location)
		if outage.StartedAt.Before(windowEnd) && outage.EndedAt.After(windowStart) {
			result[day.Format("2006-01-02")] = true
		}
	}
}

func isBusinessWeekday(date time.Time) bool {
	weekday := date.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

func sameBusinessDate(date time.Time, value time.Time, location *time.Location) bool {
	return normalizeDate(date, location).Equal(normalizeDate(value, location))
}

func positiveMinutes(duration time.Duration) int {
	if duration <= 0 {
		return 0
	}

	return int(math.Round(duration.Minutes()))
}
