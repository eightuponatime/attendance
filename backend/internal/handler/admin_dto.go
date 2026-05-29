package handler

import (
	"time"

	"attendance/internal/domain"
)

type adminAccessResponse struct {
	Email      string     `json:"email"`
	FullName   *string    `json:"full_name"`
	CreatedAt  time.Time  `json:"created_at"`
	CreatedBy  *string    `json:"created_by"`
	RevokedAt  *time.Time `json:"revoked_at"`
	IsActive   bool       `json:"is_active"`
	HasSession bool       `json:"has_session"`
}

type adminAccessListResponse struct {
	Items []adminAccessResponse `json:"items"`
}

type adminSessionResponse struct {
	Id        string     `json:"id"`
	Email     string     `json:"email"`
	FullName  string     `json:"full_name"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	IsActive  bool       `json:"is_active"`
}

type adminSessionListResponse struct {
	Items []adminSessionResponse `json:"items"`
}

type adminReportResponse struct {
	Id          string    `json:"id"`
	PeriodStart string    `json:"period_start"`
	PeriodEnd   string    `json:"period_end"`
	SentAt      time.Time `json:"sent_at"`
}

type adminReportListResponse struct {
	Items []adminReportResponse `json:"items"`
}

type adminEmployeesMonthResponse struct {
	From                string                         `json:"from"`
	To                  string                         `json:"to"`
	WorkdayStart        string                         `json:"workday_start"`
	WorkdayEnd          string                         `json:"workday_end"`
	TargetMinutesPerDay int                            `json:"target_minutes_per_day"`
	Employees           []adminEmployeeSummaryResponse `json:"employees"`
}

type adminEmployeeMonthDetailResponse struct {
	adminEmployeeSummaryResponse
	Days []attendanceDaySummaryResponse `json:"days"`
}

type adminEmployeeSummaryResponse struct {
	UserId           string `json:"user_id"`
	Email            string `json:"email"`
	FullName         string `json:"full_name"`
	WorkedMinutes    int    `json:"worked_minutes"`
	TargetMinutes    int    `json:"target_minutes"`
	CheckInCount     int    `json:"check_in_count"`
	CheckOutCount    int    `json:"check_out_count"`
	LateCount        int    `json:"late_count"`
	EarlyLeaveCount  int    `json:"early_leave_count"`
	MissingCheckOuts int    `json:"missing_check_outs"`
	WorkedDays       int    `json:"worked_days"`
}

type adminSuspiciousActivityResponse struct {
	From          string                          `json:"from"`
	To            string                          `json:"to"`
	DeviceMatches []adminSuspiciousDeviceResponse `json:"device_matches"`
	IPMatches     []adminSuspiciousIPResponse     `json:"ip_matches"`
}

type adminSystemOutageResponse struct {
	Id                   string     `json:"id"`
	StartedAt            time.Time  `json:"started_at"`
	EndedAt              time.Time  `json:"ended_at"`
	Reason               *string    `json:"reason"`
	CreatedAt            time.Time  `json:"created_at"`
	AffectedBusinessDate *string    `json:"affected_business_date"`
	ImpactsWorkHours     bool       `json:"impacts_work_hours"`
	ResolvedAt           *time.Time `json:"resolved_at"`
	ResolvedBy           *string    `json:"resolved_by"`
	ResolutionNote       *string    `json:"resolution_note"`
}

type adminSystemOutageListResponse struct {
	Items []adminSystemOutageResponse `json:"items"`
}

type adminOutageDayResponse struct {
	Outage    adminSystemOutageResponse        `json:"outage"`
	Employees []adminOutageDayEmployeeResponse `json:"employees"`
}

type adminOutageDayEmployeeResponse struct {
	UserId     string     `json:"user_id"`
	Email      string     `json:"email"`
	FullName   string     `json:"full_name"`
	CheckInAt  *time.Time `json:"check_in_at"`
	CheckOutAt *time.Time `json:"check_out_at"`
}

type adminSuspiciousDeviceResponse struct {
	DeviceId string                       `json:"device_id"`
	Owner    adminSuspiciousActorResponse `json:"owner"`
	Event    adminAttendanceEventResponse `json:"event"`
}

type adminSuspiciousIPResponse struct {
	ExternalIP     string                       `json:"external_ip"`
	Event          adminAttendanceEventResponse `json:"event"`
	PreviousEvent  adminAttendanceEventResponse `json:"previous_event"`
	MinutesBetween int                          `json:"minutes_between"`
}

type adminSuspiciousActorResponse struct {
	UserId   string `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type adminAttendanceEventResponse struct {
	EventId      string    `json:"event_id"`
	UserId       string    `json:"user_id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	BusinessDate string    `json:"business_date"`
	EventType    string    `json:"event_type"`
	EventAt      time.Time `json:"event_at"`
	DeviceId     string    `json:"device_id"`
	ExternalIP   string    `json:"external_ip"`
}

func newAdminAccessListResponse(rows []domain.AdminAccess) adminAccessListResponse {
	items := make([]adminAccessResponse, 0, len(rows))
	for _, row := range rows {
		row := row
		items = append(items, newAdminAccessResponse(&row))
	}

	return adminAccessListResponse{Items: items}
}

func newAdminAccessResponse(row *domain.AdminAccess) adminAccessResponse {
	return adminAccessResponse{
		Email:      row.Email,
		FullName:   row.FullName,
		CreatedAt:  row.CreatedAt,
		CreatedBy:  row.CreatedBy,
		RevokedAt:  row.RevokedAt,
		IsActive:   row.RevokedAt == nil,
		HasSession: row.FullName != nil && *row.FullName != "",
	}
}

func newAdminSessionListResponse(rows []domain.AdminSession) adminSessionListResponse {
	items := make([]adminSessionResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, adminSessionResponse{
			Id:        row.Id.String(),
			Email:     row.Email,
			FullName:  row.FullName,
			CreatedAt: row.CreatedAt,
			ExpiresAt: row.ExpiresAt,
			RevokedAt: row.RevokedAt,
			IsActive:  row.RevokedAt == nil && row.ExpiresAt.After(time.Now()),
		})
	}

	return adminSessionListResponse{Items: items}
}

func newAdminReportListResponse(rows []domain.AdminReportRun) adminReportListResponse {
	items := make([]adminReportResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, adminReportResponse{
			Id:          row.Id.String(),
			PeriodStart: row.PeriodStart.Format("2006-01-02"),
			PeriodEnd:   row.PeriodEnd.Format("2006-01-02"),
			SentAt:      row.SentAt,
		})
	}

	return adminReportListResponse{Items: items}
}

func newAdminEmployeesMonthResponse(
	overview *domain.AdminEmployeesMonthOverview,
) adminEmployeesMonthResponse {
	employees := make([]adminEmployeeSummaryResponse, 0, len(overview.Employees))
	for _, employee := range overview.Employees {
		employees = append(employees, newAdminEmployeeSummaryResponse(employee))
	}

	return adminEmployeesMonthResponse{
		From:                overview.From.Format("2006-01-02"),
		To:                  overview.To.Format("2006-01-02"),
		WorkdayStart:        overview.WorkdayStart,
		WorkdayEnd:          overview.WorkdayEnd,
		TargetMinutesPerDay: overview.TargetMinutesPerDay,
		Employees:           employees,
	}
}

func newAdminEmployeeMonthDetailResponse(
	summary *domain.AdminEmployeeMonthSummary,
) adminEmployeeMonthDetailResponse {
	days := make([]attendanceDaySummaryResponse, 0, len(summary.AttendanceDays))
	for _, day := range summary.AttendanceDays {
		days = append(days, newAttendanceDaySummaryResponse(day))
	}

	return adminEmployeeMonthDetailResponse{
		adminEmployeeSummaryResponse: newAdminEmployeeSummaryResponse(*summary),
		Days:                         days,
	}
}

func newAdminEmployeeSummaryResponse(
	summary domain.AdminEmployeeMonthSummary,
) adminEmployeeSummaryResponse {
	return adminEmployeeSummaryResponse{
		UserId:           summary.UserId.String(),
		Email:            summary.Email,
		FullName:         summary.FullName,
		WorkedMinutes:    summary.WorkedMinutes,
		TargetMinutes:    summary.TargetMinutes,
		CheckInCount:     summary.CheckInCount,
		CheckOutCount:    summary.CheckOutCount,
		LateCount:        summary.LateCount,
		EarlyLeaveCount:  summary.EarlyLeaveCount,
		MissingCheckOuts: summary.MissingCheckOuts,
		WorkedDays:       summary.WorkedDays,
	}
}

func newAdminSuspiciousActivityResponse(
	activity *domain.AdminSuspiciousActivity,
) adminSuspiciousActivityResponse {
	deviceMatches := make([]adminSuspiciousDeviceResponse, 0, len(activity.DeviceMatches))
	for _, match := range activity.DeviceMatches {
		deviceMatches = append(deviceMatches, adminSuspiciousDeviceResponse{
			DeviceId: match.DeviceId,
			Owner:    newAdminSuspiciousActorResponse(match.Owner),
			Event:    newAdminAttendanceEventResponse(match.Event),
		})
	}

	ipMatches := make([]adminSuspiciousIPResponse, 0, len(activity.IPMatches))
	for _, match := range activity.IPMatches {
		ipMatches = append(ipMatches, adminSuspiciousIPResponse{
			ExternalIP:     match.ExternalIp,
			Event:          newAdminAttendanceEventResponse(match.Event),
			PreviousEvent:  newAdminAttendanceEventResponse(match.PreviousEvent),
			MinutesBetween: match.MinutesBetween,
		})
	}

	return adminSuspiciousActivityResponse{
		From:          activity.From.Format("2006-01-02"),
		To:            activity.To.Format("2006-01-02"),
		DeviceMatches: deviceMatches,
		IPMatches:     ipMatches,
	}
}

func newAdminAttendanceEventResponses(
	events []domain.AdminAttendanceEventRow,
) []adminAttendanceEventResponse {
	items := make([]adminAttendanceEventResponse, 0, len(events))
	for _, event := range events {
		items = append(items, newAdminAttendanceEventResponse(event))
	}

	return items
}

func newAdminAttendanceEventResponse(event domain.AdminAttendanceEventRow) adminAttendanceEventResponse {
	return adminAttendanceEventResponse{
		EventId:      event.EventId.String(),
		UserId:       event.UserId.String(),
		Email:        event.Email,
		FullName:     event.FullName,
		BusinessDate: event.BusinessDate.Format("2006-01-02"),
		EventType:    event.EventType,
		EventAt:      event.EventAt,
		DeviceId:     event.DeviceId,
		ExternalIP:   event.ExternalIp,
	}
}

func newAdminSuspiciousActorResponse(actor domain.AdminSuspiciousActor) adminSuspiciousActorResponse {
	return adminSuspiciousActorResponse{
		UserId:   actor.UserId.String(),
		Email:    actor.Email,
		FullName: actor.FullName,
	}
}

func newAdminSystemOutageListResponse(rows []domain.SystemOutage) adminSystemOutageListResponse {
	items := make([]adminSystemOutageResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, newAdminSystemOutageResponse(row))
	}

	return adminSystemOutageListResponse{Items: items}
}

func newAdminSystemOutageResponse(row domain.SystemOutage) adminSystemOutageResponse {
	var affected *string
	if row.AffectedBusinessDate != nil {
		value := row.AffectedBusinessDate.Format("2006-01-02")
		affected = &value
	}

	return adminSystemOutageResponse{
		Id:                   row.Id.String(),
		StartedAt:            row.StartedAt,
		EndedAt:              row.EndedAt,
		Reason:               row.Reason,
		CreatedAt:            row.CreatedAt,
		AffectedBusinessDate: affected,
		ImpactsWorkHours:     row.ImpactsWorkHours,
		ResolvedAt:           row.ResolvedAt,
		ResolvedBy:           row.ResolvedBy,
		ResolutionNote:       row.ResolutionNote,
	}
}

func newAdminOutageDayResponse(
	outage *domain.SystemOutage,
	rows []domain.AdminOutageDayEmployeeRow,
) adminOutageDayResponse {
	employees := make([]adminOutageDayEmployeeResponse, 0, len(rows))
	for _, row := range rows {
		employees = append(employees, adminOutageDayEmployeeResponse{
			UserId:     row.UserId.String(),
			Email:      row.Email,
			FullName:   row.FullName,
			CheckInAt:  row.CheckInAt,
			CheckOutAt: row.CheckOutAt,
		})
	}

	return adminOutageDayResponse{
		Outage:    newAdminSystemOutageResponse(*outage),
		Employees: employees,
	}
}
