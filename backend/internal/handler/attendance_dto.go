package handler

import (
	"time"

	"attendance/internal/domain"
)

type attendanceEventResponse struct {
	EventType string    `json:"event_type"`
	EventAt   time.Time `json:"event_at"`
	Status    string    `json:"status"`
}

type attendanceTodayResponse struct {
	BusinessDate      string                   `json:"business_date"`
	CheckIn           *attendanceEventResponse `json:"check_in"`
	CheckOut          *attendanceEventResponse `json:"check_out"`
	LateMinutes       int                      `json:"late_minutes"`
	EarlyLeaveMinutes int                      `json:"early_leave_minutes"`
	ImpactedByOutage  bool                     `json:"impacted_by_outage"`
	CanCheckIn        bool                     `json:"can_check_in"`
	CanCheckOut       bool                     `json:"can_check_out"`
}

type attendanceSummaryResponse struct {
	From                string                         `json:"from"`
	To                  string                         `json:"to"`
	WorkdayStart        string                         `json:"workday_start"`
	WorkdayEnd          string                         `json:"workday_end"`
	TargetMinutesPerDay int                            `json:"target_minutes_per_day"`
	Days                []attendanceDaySummaryResponse `json:"days"`
}

type attendanceDaySummaryResponse struct {
	Date              string                          `json:"date"`
	CheckInAt         *string                         `json:"check_in_at"`
	CheckOutAt        *string                         `json:"check_out_at"`
	WorkedMinutes     int                             `json:"worked_minutes"`
	LateMinutes       int                             `json:"late_minutes"`
	EarlyLeaveMinutes int                             `json:"early_leave_minutes"`
	Status            string                          `json:"status"`
	ImpactedByOutage  bool                            `json:"impacted_by_outage"`
	Explanations      []attendanceExplanationResponse `json:"explanations"`
}

type attendanceExplanationResponse struct {
	Id                   string     `json:"id"`
	BusinessDate         string     `json:"business_date"`
	ReasonType           string     `json:"reason_type"`
	Comment              string     `json:"comment"`
	Status               string     `json:"status"`
	ReviewedByAdminEmail *string    `json:"reviewed_by_admin_email"`
	ReviewedAt           *time.Time `json:"reviewed_at"`
	ReviewNote           *string    `json:"review_note"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func newAttendanceTodayResponse(today *domain.AttendanceToday) attendanceTodayResponse {
	response := attendanceTodayResponse{
		BusinessDate:      today.BusinessDate.Format("2006-01-02"),
		LateMinutes:       today.LateMinutes,
		EarlyLeaveMinutes: today.EarlyLeaveMinutes,
		ImpactedByOutage:  today.ImpactedByOutage,
		CanCheckIn:        today.CheckIn == nil,
		CanCheckOut:       today.CheckIn != nil && today.CheckOut == nil,
	}

	if today.CheckIn != nil {
		response.CheckIn = newAttendanceEventResponse(today.CheckIn)
	}
	if today.CheckOut != nil {
		response.CheckOut = newAttendanceEventResponse(today.CheckOut)
	}

	return response
}

func newAttendanceEventResponse(event *domain.AttendanceEvents) *attendanceEventResponse {
	return &attendanceEventResponse{
		EventType: event.EventType,
		EventAt:   event.EventAt,
		Status:    event.Status,
	}
}

func newAttendanceSummaryResponse(summary *domain.AttendanceSummary) attendanceSummaryResponse {
	days := make([]attendanceDaySummaryResponse, 0, len(summary.Days))
	for _, day := range summary.Days {
		days = append(days, newAttendanceDaySummaryResponse(day))
	}

	return attendanceSummaryResponse{
		From:                summary.From.Format("2006-01-02"),
		To:                  summary.To.Format("2006-01-02"),
		WorkdayStart:        summary.WorkdayStart,
		WorkdayEnd:          summary.WorkdayEnd,
		TargetMinutesPerDay: summary.TargetMinutesPerDay,
		Days:                days,
	}
}

func newAttendanceDaySummaryResponse(day domain.AttendanceDaySummary) attendanceDaySummaryResponse {
	explanations := make([]attendanceExplanationResponse, 0, len(day.Explanations))
	for _, explanation := range day.Explanations {
		explanations = append(explanations, newAttendanceExplanationResponse(explanation))
	}

	return attendanceDaySummaryResponse{
		Date:              day.Date.Format("2006-01-02"),
		CheckInAt:         formatOptionalTime(day.CheckInAt),
		CheckOutAt:        formatOptionalTime(day.CheckOutAt),
		WorkedMinutes:     day.WorkedMinutes,
		LateMinutes:       day.LateMinutes,
		EarlyLeaveMinutes: day.EarlyLeaveMinutes,
		Status:            day.Status,
		ImpactedByOutage:  day.ImpactedByOutage,
		Explanations:      explanations,
	}
}

func newAttendanceExplanationResponse(explanation domain.AttendanceExplanation) attendanceExplanationResponse {
	return attendanceExplanationResponse{
		Id:                   explanation.Id.String(),
		BusinessDate:         explanation.BusinessDate.Format("2006-01-02"),
		ReasonType:           explanation.ReasonType,
		Comment:              explanation.Comment,
		Status:               explanation.Status,
		ReviewedByAdminEmail: explanation.ReviewedByAdminEmail,
		ReviewedAt:           explanation.ReviewedAt,
		ReviewNote:           explanation.ReviewNote,
		CreatedAt:            explanation.CreatedAt,
		UpdatedAt:            explanation.UpdatedAt,
	}
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := value.Format(time.RFC3339)
	return &formatted
}
