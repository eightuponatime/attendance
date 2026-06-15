package impl

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"time"

	"attendance/config"
	"attendance/internal/domain"
	"attendance/internal/service"

	"github.com/google/uuid"
)

type ExcelService struct {
	adminService service.AdminService
	cfg          *config.Config
}

func NewExcelService(adminService service.AdminService, cfg *config.Config) *ExcelService {
	return &ExcelService{
		adminService: adminService,
		cfg:          cfg,
	}
}

func (s *ExcelService) BuildMonthlyReport(
	ctx context.Context,
	from time.Time,
	to time.Time,
) (*domain.ExcelReportFile, error) {
	employees, err := s.adminService.EmployeesMonth(ctx, from, to)
	if err != nil {
		return nil, err
	}
	suspicious, err := s.adminService.SuspiciousActivity(ctx, from, to)
	if err != nil {
		return nil, err
	}

	suspiciousCounts := suspiciousCountsByUser(suspicious)
	details, err := s.employeeDetails(ctx, employees, from, to)
	if err != nil {
		return nil, err
	}

	reportScope, err := s.reportScope(employees)
	if err != nil {
		return nil, err
	}

	data, err := buildExcelXML(employees, details, suspiciousCounts, reportScope)
	if err != nil {
		return nil, err
	}

	return &domain.ExcelReportFile{
		Filename:    excelFilename(from, to),
		ContentType: "application/vnd.ms-excel; charset=utf-8",
		Data:        data,
	}, nil
}

func (s *ExcelService) employeeDetails(
	ctx context.Context,
	overview *domain.AdminEmployeesMonthOverview,
	from time.Time,
	to time.Time,
) ([]domain.AdminEmployeeMonthSummary, error) {
	if overview == nil {
		return nil, nil
	}

	details := make([]domain.AdminEmployeeMonthSummary, 0, len(overview.Employees))
	for _, employee := range overview.Employees {
		detail, err := s.adminService.EmployeeMonth(ctx, employee.UserId, from, to)
		if err != nil {
			return nil, err
		}
		if detail == nil {
			continue
		}
		details = append(details, *detail)
	}

	return details, nil
}

type excelReportScope struct {
	PlannedThrough   time.Time
	FinalizedThrough time.Time
}

func (s *ExcelService) reportScope(overview *domain.AdminEmployeesMonthOverview) (excelReportScope, error) {
	location, err := time.LoadLocation(s.cfg.BusinessTimezone)
	if err != nil {
		return excelReportScope{}, err
	}

	now := time.Now().In(location)
	today := dateOnly(now)
	plannedThrough := minDate(overview.To, today)
	finalizedThrough := minDate(overview.To, finalizedBusinessDate(now, overview.WorkdayEnd))

	return excelReportScope{
		PlannedThrough:   plannedThrough,
		FinalizedThrough: finalizedThrough,
	}, nil
}

func finalizedBusinessDate(now time.Time, workdayEnd string) time.Time {
	endAt, err := time.ParseInLocation("15:04", workdayEnd, now.Location())
	if err != nil {
		return dateOnly(now.AddDate(0, 0, -1))
	}

	todayEnd := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		endAt.Hour(),
		endAt.Minute(),
		0,
		0,
		now.Location(),
	)
	if now.Before(todayEnd) {
		return dateOnly(now.AddDate(0, 0, -1))
	}
	return dateOnly(now)
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func minDate(left time.Time, right time.Time) time.Time {
	leftDate := dateOnly(left)
	rightDate := dateOnly(right)
	if leftDate.After(rightDate) {
		return rightDate
	}
	return leftDate
}

func suspiciousCountsByUser(activity *domain.AdminSuspiciousActivity) map[uuid.UUID]int {
	counts := make(map[uuid.UUID]int)
	if activity == nil {
		return counts
	}

	for _, match := range activity.DeviceMatches {
		counts[match.Owner.UserId]++
		counts[match.Event.UserId]++
	}
	for _, match := range activity.IPMatches {
		counts[match.Event.UserId]++
		counts[match.PreviousEvent.UserId]++
	}

	return counts
}

func buildExcelXML(
	overview *domain.AdminEmployeesMonthOverview,
	details []domain.AdminEmployeeMonthSummary,
	suspiciousCounts map[uuid.UUID]int,
	scope excelReportScope,
) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buffer.WriteString(`<?mso-application progid="Excel.Sheet"?>` + "\n")
	buffer.WriteString(`<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet" `)
	buffer.WriteString(`xmlns:o="urn:schemas-microsoft-com:office:office" `)
	buffer.WriteString(`xmlns:x="urn:schemas-microsoft-com:office:excel" `)
	buffer.WriteString(`xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet">`)
	buffer.WriteString(`<Styles>`)
	buffer.WriteString(`<Style ss:ID="Header"><Font ss:Bold="1"/><Interior ss:Color="#DCEAF7" ss:Pattern="Solid"/></Style>`)
	buffer.WriteString(`<Style ss:ID="Title"><Font ss:Bold="1" ss:Size="14"/></Style>`)
	buffer.WriteString(`</Styles>`)

	writeSummarySheet(&buffer, overview, details, suspiciousCounts, scope)

	buffer.WriteString(`</Workbook>`)
	return buffer.Bytes(), nil
}

func writeSummarySheet(
	buffer *bytes.Buffer,
	overview *domain.AdminEmployeesMonthOverview,
	details []domain.AdminEmployeeMonthSummary,
	suspiciousCounts map[uuid.UUID]int,
	scope excelReportScope,
) {
	buffer.WriteString(`<Worksheet ss:Name="Сводка"><Table>`)
	writeRow(buffer, []excelCell{
		textCell(fmt.Sprintf("Сводка за %s", excelPeriodTitle(overview.From, overview.To))),
	}, "Title")
	writeRow(buffer, []excelCell{
		textCell("Период"),
		textCell(fmt.Sprintf("%s - %s", overview.From.Format("02.01.2006"), overview.To.Format("02.01.2006"))),
	}, "")
	writeRow(buffer, nil, "")

	writeRow(buffer, []excelCell{
		textCell("ФИО"),
		textCell("Email"),
		textCell("Отработано"),
		textCell("Цель"),
		textCell("Рабочих дней"),
		textCell("Дней с отметками"),
		textCell("Опозданий"),
		textCell("Ранних уходов"),
		textCell("Дней без ухода"),
		textCell("Подозрительных событий"),
		textCell("Пропущено рабочих дней"),
	}, "Header")

	employees := append([]domain.AdminEmployeeMonthSummary(nil), details...)
	sort.SliceStable(employees, func(i int, j int) bool {
		return employees[i].FullName < employees[j].FullName
	})

	for _, employee := range employees {
		plannedWorkdays := workdayCount(employee.AttendanceDays, scope.PlannedThrough)
		missingCheckOuts := missingCheckOutCount(employee.AttendanceDays, scope.FinalizedThrough)
		missedWorkdays := missedWorkdayCount(employee.AttendanceDays, scope.FinalizedThrough)
		targetMinutes := plannedWorkdays * overview.TargetMinutesPerDay

		writeRow(buffer, []excelCell{
			textCell(employee.FullName),
			textCell(employee.Email),
			textCell(minutesToHoursText(employee.WorkedMinutes)),
			textCell(minutesToHoursText(targetMinutes)),
			numberCell(plannedWorkdays),
			numberCell(employee.WorkedDays),
			numberCell(employee.LateCount),
			numberCell(employee.EarlyLeaveCount),
			numberCell(missingCheckOuts),
			numberCell(suspiciousCounts[employee.UserId]),
			numberCell(missedWorkdays),
		}, "")
	}

	buffer.WriteString(`</Table></Worksheet>`)
}

func workdayCount(days []domain.AttendanceDaySummary, through time.Time) int {
	count := 0
	for _, day := range days {
		if isReportableWorkday(day.Date, through) {
			count++
		}
	}
	return count
}

func missedWorkdayCount(days []domain.AttendanceDaySummary, through time.Time) int {
	count := 0
	for _, day := range days {
		if isReportableWorkday(day.Date, through) && day.CheckInAt == nil {
			count++
		}
	}
	return count
}

func missingCheckOutCount(days []domain.AttendanceDaySummary, through time.Time) int {
	count := 0
	for _, day := range days {
		if isReportableWorkday(day.Date, through) && day.CheckInAt != nil && day.CheckOutAt == nil {
			count++
		}
	}
	return count
}

func isReportableWorkday(date time.Time, through time.Time) bool {
	return isWorkday(date) && !dateOnly(date).After(dateOnly(through))
}

func isWorkday(date time.Time) bool {
	weekday := date.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

type excelCell struct {
	Value string
	Type  string
}

func textCell(value string) excelCell {
	return excelCell{Value: value, Type: "String"}
}

func numberCell(value int) excelCell {
	return excelCell{Value: fmt.Sprintf("%d", value), Type: "Number"}
}

func writeRow(buffer *bytes.Buffer, cells []excelCell, style string) {
	buffer.WriteString(`<Row>`)
	for _, cell := range cells {
		if style != "" {
			buffer.WriteString(`<Cell ss:StyleID="`)
			writeEscaped(buffer, style)
			buffer.WriteString(`">`)
		} else {
			buffer.WriteString(`<Cell>`)
		}
		buffer.WriteString(`<Data ss:Type="`)
		writeEscaped(buffer, cell.Type)
		buffer.WriteString(`">`)
		writeEscaped(buffer, cell.Value)
		buffer.WriteString(`</Data></Cell>`)
	}
	buffer.WriteString(`</Row>`)
}

func writeEscaped(buffer *bytes.Buffer, value string) {
	_ = xml.EscapeText(buffer, []byte(value))
}

func minutesToHoursText(minutes int) string {
	hours := minutes / 60
	rest := minutes % 60
	if rest == 0 {
		return fmt.Sprintf("%d ч", hours)
	}
	return fmt.Sprintf("%d ч %d мин", hours, rest)
}

func excelFilename(from time.Time, to time.Time) string {
	return fmt.Sprintf(
		"Посещаемость-%s-%s.xls",
		excelFilenameDate(from),
		excelFilenameDate(to),
	)
}

func excelFilenameDate(value time.Time) string {
	return fmt.Sprintf("%d%s%d", value.Day(), excelMonthGenitive(value.Month()), value.Year())
}

func excelPeriodTitle(from time.Time, to time.Time) string {
	if isExcelFullCalendarMonth(from, to) {
		return fmt.Sprintf("%s %d", excelMonthNominative(from.Month()), from.Year())
	}
	if from.Year() == to.Year() && from.Month() == to.Month() {
		return fmt.Sprintf("%d-%d %s %d", from.Day(), to.Day(), excelMonthGenitive(from.Month()), from.Year())
	}
	return fmt.Sprintf(
		"%d %s %d - %d %s %d",
		from.Day(),
		excelMonthGenitive(from.Month()),
		from.Year(),
		to.Day(),
		excelMonthGenitive(to.Month()),
		to.Year(),
	)
}

func isExcelFullCalendarMonth(from time.Time, to time.Time) bool {
	monthStart := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
	monthEnd := monthStart.AddDate(0, 1, -1)
	return sameExcelDate(from, monthStart) && sameExcelDate(to, monthEnd)
}

func sameExcelDate(left time.Time, right time.Time) bool {
	return left.Year() == right.Year() && left.Month() == right.Month() && left.Day() == right.Day()
}

func excelMonthNominative(month time.Month) string {
	months := [...]string{
		"январь",
		"февраль",
		"март",
		"апрель",
		"май",
		"июнь",
		"июль",
		"август",
		"сентябрь",
		"октябрь",
		"ноябрь",
		"декабрь",
	}
	return months[int(month)-1]
}

func excelMonthGenitive(month time.Month) string {
	months := [...]string{
		"января",
		"февраля",
		"марта",
		"апреля",
		"мая",
		"июня",
		"июля",
		"августа",
		"сентября",
		"октября",
		"ноября",
		"декабря",
	}
	return months[int(month)-1]
}
