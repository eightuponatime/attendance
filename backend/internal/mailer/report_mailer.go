package mailer

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html"
	"mime"
	"net"
	"net/smtp"
	"sort"
	"strings"
	"time"

	"attendance/config"
	"attendance/internal/domain"
	"attendance/internal/service"

	"go.uber.org/zap"
)

type ReportMailer struct {
	cfg          *config.Config
	adminService service.AdminService
	excelService service.ExcelService
	logger       *zap.SugaredLogger
}

func NewReportMailer(
	cfg *config.Config,
	adminService service.AdminService,
	excelService service.ExcelService,
	logger *zap.SugaredLogger,
) *ReportMailer {
	return &ReportMailer{
		cfg:          cfg,
		adminService: adminService,
		excelService: excelService,
		logger:       logger,
	}
}

func (m *ReportMailer) Start(ctx context.Context) {
	if !m.cfg.ReportEmailEnabled || !m.cfg.SMTPConfigured() {
		m.logger.Infow(
			"report email sender disabled",
			"enabled", m.cfg.ReportEmailEnabled,
			"smtpConfigured", m.cfg.SMTPConfigured(),
		)
		return
	}

	go m.loop(ctx)
	go func() {
		if err := m.SendLatestDueMonthlyReport(ctx); err != nil {
			m.logger.Errorf("failed to send due admin report email: %v", err)
		}
	}()
}

func (m *ReportMailer) loop(ctx context.Context) {
	for {
		wait, err := m.durationUntilNextRun()
		if err != nil {
			m.logger.Errorf("report email schedule is invalid: %v", err)
			return
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := m.SendLatestDueMonthlyReport(ctx); err != nil {
				m.logger.Errorf("failed to send admin report email: %v", err)
			}
		}
	}
}

func (m *ReportMailer) SendCurrentMonthReport(ctx context.Context) error {
	return m.SendLatestDueMonthlyReport(ctx)
}

func (m *ReportMailer) SendLatestDueMonthlyReport(ctx context.Context) error {
	if !m.cfg.SMTPConfigured() {
		return fmt.Errorf("smtp is not configured")
	}

	from, to, periodTitle, displayPeriod, due, err := m.latestDueReportRange()
	if err != nil {
		return err
	}
	if !due {
		m.logger.Info("report email skipped: no closed period is due yet")
		return nil
	}

	reportRun, err := m.adminService.GetReportRunByPeriod(ctx, from, to)
	if err != nil {
		return err
	}
	if reportRun != nil {
		m.logger.Infow("report email skipped: period already sent", "period", periodTitle)
		return nil
	}

	access, err := m.adminService.ListAccess(ctx)
	if err != nil {
		return err
	}
	recipients := activeAdminEmails(access)
	if len(recipients) == 0 {
		m.logger.Info("report email skipped: no active admin recipients")
		return nil
	}

	employees, err := m.adminService.EmployeesMonth(ctx, from, to)
	if err != nil {
		return err
	}
	suspicious, err := m.adminService.SuspiciousActivity(ctx, from, to)
	if err != nil {
		return err
	}

	stats := reportStats(employees, suspicious)
	rows, err := m.reportEmployeeRows(ctx, employees, suspicious, from, to)
	if err != nil {
		return err
	}
	subject := fmt.Sprintf("HR отчет за %s", displayPeriod)
	body := m.reportHTML(displayPeriod, stats, rows)
	attachments, err := m.reportAttachments(ctx, from, to)
	if err != nil {
		return err
	}

	for _, recipient := range recipients {
		if err := m.send(recipient, subject, body, attachments...); err != nil {
			return fmt.Errorf("send to %s: %w", recipient, err)
		}
	}

	if _, err := m.adminService.CreateReportRun(ctx, domain.CreateAdminReportRunInput{
		PeriodStart: from,
		PeriodEnd:   to,
	}); err != nil {
		return err
	}

	m.logger.Infow("admin report email sent", "recipients", len(recipients), "period", periodTitle)
	return nil
}

func (m *ReportMailer) reportAttachments(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]emailAttachment, error) {
	if m.excelService == nil {
		return nil, nil
	}

	file, err := m.excelService.BuildMonthlyReport(ctx, from, to)
	if err != nil {
		return nil, err
	}

	return []emailAttachment{{
		Filename:    file.Filename,
		ContentType: file.ContentType,
		Data:        file.Data,
	}}, nil
}

func (m *ReportMailer) reportHTML(periodTitle string, stats adminReportStats, rows []adminReportEmployeeRow) string {
	adminURL := strings.TrimRight(m.cfg.AdminFrontendURLValue(), "/") + "/admin"
	return fmt.Sprintf(`<!doctype html>
<html>
<body style="margin:0;padding:0;background:#f3f6fa;font-family:Arial,sans-serif;color:#071026;">
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#f3f6fa;padding:28px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:760px;border-collapse:separate;border-spacing:0;">
          <tr>
            <td style="padding:28px 30px;border-radius:14px;background:#113650;color:#ffffff;">
              <div style="margin:0 0 12px;color:#c6d4df;font-size:12px;font-weight:700;letter-spacing:1.6px;text-transform:uppercase;">Панель HR</div>
              <div style="margin:0 0 10px;font-size:28px;line-height:1.2;font-weight:800;">Сводка за %s</div>
              <div style="margin:0;color:#e5edf3;font-size:14px;line-height:1.45;">Автоматически сформированная сводка по посещаемости сотрудников.</div>
            </td>
          </tr>
          <tr><td style="height:16px;"></td></tr>
          <tr>
            <td style="padding:18px 20px;border:1px solid #dfe7f1;border-radius:14px;background:#ffffff;">
              <div style="margin:0 0 12px;color:#071026;font-size:20px;line-height:1.25;font-weight:800;">Ключевые показатели</div>
              <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:separate;border-spacing:8px;">
                %s
              </table>
            </td>
          </tr>
          <tr><td style="height:16px;"></td></tr>
          <tr>
            <td style="padding:18px 20px;border:1px solid #b9dcff;border-radius:14px;background:#eaf6ff;">
              <table role="presentation" width="100%%" cellspacing="0" cellpadding="0">
                <tr>
                  <td style="padding:0 16px 0 0;">
                    <div style="margin:0 0 6px;color:#07598f;font-size:19px;line-height:1.25;font-weight:800;">Админ-панель</div>
                    <div style="margin:0;color:#2e5d7b;font-size:13px;line-height:1.45;">Детали сотрудников, календарь, сбои сервера и обработка заявок.</div>
                  </td>
                  <td align="right" style="width:170px;">
                    <a href="%s" style="display:inline-block;padding:12px 16px;border-radius:8px;background:#113650;color:#ffffff;text-decoration:none;font-size:14px;font-weight:800;white-space:nowrap;">Открыть панель</a>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <tr><td style="height:16px;"></td></tr>
          <tr>
            <td style="padding:22px 24px;border:1px solid #dfe7f1;border-radius:14px;background:#ffffff;">
              <div style="margin:0 0 8px;color:#071026;font-size:21px;line-height:1.25;font-weight:800;">Сотрудники</div>
              <div style="margin:0 0 16px;color:#697083;font-size:13px;line-height:1.45;">Краткая таблица продублирована в письме. Полный Excel-файл приложен к этому письму.</div>
              %s
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		html.EscapeString(periodTitle),
		statGrid(stats),
		html.EscapeString(adminURL),
		employeeTable(rows),
	)
}

func statGrid(stats adminReportStats) string {
	return fmt.Sprintf(
		`<tr>%s%s</tr><tr>%s%s</tr>`,
		statCell("Сотрудников", stats.Employees),
		statCell("С опозданиями", stats.LateEmployees),
		statCell("С ранними уходами", stats.EarlyLeaveEmployees),
		statCell("С подозрительной активностью", stats.SuspiciousEmployees),
	)
}

func statCell(label string, value int) string {
	return fmt.Sprintf(
		`<td width="50%%" style="padding:12px 14px;border-radius:9px;background:#f6f8fb;">
                  <div style="margin:0 0 5px;color:#697083;font-size:13px;line-height:1.25;">%s</div>
                  <div style="margin:0;color:#071026;font-size:24px;line-height:1;font-weight:800;">%d</div>
                </td>`,
		html.EscapeString(label),
		value,
	)
}

func employeeTable(rows []adminReportEmployeeRow) string {
	if len(rows) == 0 {
		return `<div style="padding:14px 16px;border-radius:8px;background:#f6f8fb;color:#697083;font-size:14px;">За период нет сотрудников для отображения.</div>`
	}

	var builder strings.Builder
	builder.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;font-size:12px;line-height:1.35;">`)
	builder.WriteString(`<tr>`)
	for _, header := range []string{"Сотрудник", "Часы", "Дни", "Опозд.", "Ранн.", "Без ухода", "Пропущ.", "Подозр."} {
		builder.WriteString(`<th align="left" style="padding:10px 8px;border-bottom:1px solid #dfe7f1;color:#697083;font-size:11px;font-weight:800;text-transform:uppercase;">`)
		builder.WriteString(html.EscapeString(header))
		builder.WriteString(`</th>`)
	}
	builder.WriteString(`</tr>`)

	for _, row := range rows {
		builder.WriteString(`<tr>`)
		builder.WriteString(`<td style="padding:11px 8px;border-bottom:1px solid #edf1f6;color:#071026;font-weight:800;">`)
		builder.WriteString(html.EscapeString(row.FullName))
		builder.WriteString(`<div style="margin-top:2px;color:#697083;font-size:11px;font-weight:400;">`)
		builder.WriteString(html.EscapeString(row.Email))
		builder.WriteString(`</div></td>`)
		builder.WriteString(employeeNumberCell(row.HoursText, hoursTone(row.WorkedMinutes, row.TargetMinutes)))
		builder.WriteString(employeeNumberCell(fmt.Sprintf("%d", row.WorkedDays)))
		builder.WriteString(employeeNumberCell(fmt.Sprintf("%d", row.LateCount), issueTone(row.LateCount)))
		builder.WriteString(employeeNumberCell(fmt.Sprintf("%d", row.EarlyLeaveCount), issueTone(row.EarlyLeaveCount)))
		builder.WriteString(employeeNumberCell(fmt.Sprintf("%d", row.MissingCheckOuts), issueTone(row.MissingCheckOuts)))
		builder.WriteString(employeeNumberCell(fmt.Sprintf("%d", row.MissedWorkdays), issueTone(row.MissedWorkdays)))
		builder.WriteString(employeeNumberCell(fmt.Sprintf("%d", row.SuspiciousCount)))
		builder.WriteString(`</tr>`)
	}

	builder.WriteString(`</table>`)
	return builder.String()
}

func employeeNumberCell(value string, tone ...string) string {
	cellTone := ""
	if len(tone) > 0 {
		cellTone = tone[0]
	}
	style := "padding:11px 8px;border-bottom:1px solid #edf1f6;color:#071026;font-weight:700;white-space:nowrap;"
	switch cellTone {
	case "good":
		style += "background:#dcfce7;color:#166534;"
	case "bad":
		style += "background:#fee2e2;color:#991b1b;"
	}
	return fmt.Sprintf(
		`<td align="right" style="%s">%s</td>`,
		style,
		html.EscapeString(value),
	)
}

func issueTone(value int) string {
	if value > 0 {
		return "bad"
	}
	return ""
}

func hoursTone(workedMinutes int, targetMinutes int) string {
	if workedMinutes >= targetMinutes {
		return "good"
	}
	return "bad"
}

func reportPeriodTitle(from time.Time, to time.Time) string {
	if isFullCalendarMonth(from, to) {
		return fmt.Sprintf("%s %d", monthNominative(from.Month()), from.Year())
	}
	if from.Year() == to.Year() && from.Month() == to.Month() {
		return fmt.Sprintf("%d-%d %s %d", from.Day(), to.Day(), monthGenitive(from.Month()), from.Year())
	}
	return fmt.Sprintf(
		"%d %s %d - %d %s %d",
		from.Day(),
		monthGenitive(from.Month()),
		from.Year(),
		to.Day(),
		monthGenitive(to.Month()),
		to.Year(),
	)
}

func isFullCalendarMonth(from time.Time, to time.Time) bool {
	monthStart := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())
	monthEnd := monthStart.AddDate(0, 1, -1)
	return sameDate(from, monthStart) && sameDate(to, monthEnd)
}

func sameDate(left time.Time, right time.Time) bool {
	return left.Year() == right.Year() && left.Month() == right.Month() && left.Day() == right.Day()
}

func monthNominative(month time.Month) string {
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

func monthGenitive(month time.Month) string {
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

type emailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

func (m *ReportMailer) send(to string, subject string, htmlBody string, attachments ...emailAttachment) error {
	message := buildEmailMessage(m.cfg.SMTPFrom, to, subject, htmlBody, attachments...)

	addr := net.JoinHostPort(m.cfg.SMTPHost, m.cfg.SMTPPort)
	if m.cfg.SMTPPort == "465" {
		return m.sendTLS(addr, to, message)
	}

	return m.sendStartTLS(addr, to, message)
}

func buildEmailMessage(from string, to string, subject string, htmlBody string, attachments ...emailAttachment) []byte {
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      mime.QEncoding.Encode("UTF-8", subject),
		"MIME-Version": "1.0",
	}
	if len(attachments) == 0 {
		headers["Content-Type"] = `text/html; charset="UTF-8"`
	}

	var message strings.Builder
	for key, value := range headers {
		message.WriteString(key)
		message.WriteString(": ")
		message.WriteString(value)
		message.WriteString("\r\n")
	}
	message.WriteString("\r\n")

	if len(attachments) == 0 {
		message.WriteString(htmlBody)
		return []byte(message.String())
	}

	boundary := "attendance-report-boundary"
	message.WriteString("--")
	message.WriteString(boundary)
	message.WriteString("\r\n")
	message.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	message.WriteString(htmlBody)
	message.WriteString("\r\n")

	for _, attachment := range attachments {
		message.WriteString("--")
		message.WriteString(boundary)
		message.WriteString("\r\n")
		message.WriteString("Content-Type: ")
		message.WriteString(attachment.ContentType)
		message.WriteString("; name=\"")
		message.WriteString(escapeHeaderFilename(attachment.Filename))
		message.WriteString("\"\r\n")
		message.WriteString("Content-Transfer-Encoding: base64\r\n")
		message.WriteString("Content-Disposition: attachment; filename=\"")
		message.WriteString(escapeHeaderFilename(attachment.Filename))
		message.WriteString("\"\r\n\r\n")
		message.WriteString(base64Lines(attachment.Data))
		message.WriteString("\r\n")
	}

	message.WriteString("--")
	message.WriteString(boundary)
	message.WriteString("--\r\n")

	result := message.String()
	contentType := fmt.Sprintf("Content-Type: multipart/mixed; boundary=%q\r\n", boundary)
	return []byte(strings.Replace(result, "MIME-Version: 1.0\r\n", "MIME-Version: 1.0\r\n"+contentType, 1))
}

func escapeHeaderFilename(filename string) string {
	return strings.NewReplacer("\\", "_", `"`, "_", "\r", "_", "\n", "_").Replace(filename)
}

func base64Lines(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var builder strings.Builder
	for len(encoded) > 76 {
		builder.WriteString(encoded[:76])
		builder.WriteString("\r\n")
		encoded = encoded[76:]
	}
	builder.WriteString(encoded)
	return builder.String()
}

func (m *ReportMailer) sendStartTLS(addr string, to string, message []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if m.cfg.SMTPStartTLS {
		if err := client.StartTLS(&tls.Config{ServerName: m.cfg.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}

	return m.finishSMTP(client, to, message)
}

func (m *ReportMailer) sendTLS(addr string, to string, message []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.cfg.SMTPHost, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.cfg.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Close()

	return m.finishSMTP(client, to, message)
}

func (m *ReportMailer) finishSMTP(client *smtp.Client, to string, message []byte) error {
	if m.cfg.SMTPUser != "" {
		auth := smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPassword, m.cfg.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(m.cfg.SMTPFrom); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func (m *ReportMailer) latestDueReportRange() (time.Time, time.Time, string, string, bool, error) {
	switch strings.ToLower(strings.TrimSpace(m.cfg.ReportEmailFrequency)) {
	case "weekly", "week":
		return m.latestDueWeeklyRange()
	default:
		return m.latestDueMonthlyRange()
	}
}

func (m *ReportMailer) latestDueMonthlyRange() (time.Time, time.Time, string, string, bool, error) {
	location, err := time.LoadLocation(m.cfg.BusinessTimezone)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", false, err
	}

	now := time.Now().In(location)
	clock, err := time.Parse("15:04", m.cfg.ReportEmailTime)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", false, err
	}

	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
	dueAt := time.Date(now.Year(), now.Month(), 1, clock.Hour(), clock.Minute(), 0, 0, location)
	if now.Before(dueAt) {
		return time.Time{}, time.Time{}, "", "", false, nil
	}

	from := currentMonthStart.AddDate(0, -1, 0)
	to := currentMonthStart.AddDate(0, 0, -1)
	title := from.Format("2006-01")
	return from, to, title, reportPeriodTitle(from, to), true, nil
}

func (m *ReportMailer) latestDueWeeklyRange() (time.Time, time.Time, string, string, bool, error) {
	location, err := time.LoadLocation(m.cfg.BusinessTimezone)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", false, err
	}

	now := time.Now().In(location)
	clock, err := time.Parse("15:04", m.cfg.ReportEmailTime)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", false, err
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	daysSinceMonday := (int(now.Weekday()) + 6) % 7
	currentWeekStart := today.AddDate(0, 0, -daysSinceMonday)

	reportWeekStart := currentWeekStart
	if now.Before(weeklyDueAt(currentWeekStart, clock, location)) {
		reportWeekStart = currentWeekStart.AddDate(0, 0, -7)
	}

	from := reportWeekStart
	to := reportWeekStart.AddDate(0, 0, 4)
	title := fmt.Sprintf("%s_%s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	return from, to, title, reportPeriodTitle(from, to), true, nil
}

func (m *ReportMailer) durationUntilNextRun() (time.Duration, error) {
	location, err := time.LoadLocation(m.cfg.BusinessTimezone)
	if err != nil {
		return 0, err
	}

	clock, err := time.Parse("15:04", m.cfg.ReportEmailTime)
	if err != nil {
		return 0, err
	}

	if strings.EqualFold(strings.TrimSpace(m.cfg.ReportEmailFrequency), "weekly") {
		return durationUntilNextWeeklyRun(location, clock), nil
	}

	now := time.Now().In(location)
	next := time.Date(now.Year(), now.Month(), 1, clock.Hour(), clock.Minute(), 0, 0, location)
	if !next.After(now) {
		next = next.AddDate(0, 1, 0)
	}

	return time.Until(next), nil
}

func durationUntilNextWeeklyRun(location *time.Location, clock time.Time) time.Duration {
	now := time.Now().In(location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	daysSinceMonday := (int(now.Weekday()) + 6) % 7
	currentWeekStart := today.AddDate(0, 0, -daysSinceMonday)
	next := weeklyDueAt(currentWeekStart, clock, location)
	if !next.After(now) {
		next = next.AddDate(0, 0, 7)
	}
	return time.Until(next)
}

func weeklyDueAt(weekStart time.Time, clock time.Time, location *time.Location) time.Time {
	friday := weekStart.AddDate(0, 0, 4)
	return time.Date(
		friday.Year(),
		friday.Month(),
		friday.Day(),
		clock.Hour(),
		clock.Minute(),
		0,
		0,
		location,
	)
}

type adminReportStats struct {
	Employees           int
	LateEmployees       int
	EarlyLeaveEmployees int
	SuspiciousEmployees int
}

type adminReportEmployeeRow struct {
	FullName         string
	Email            string
	HoursText        string
	WorkedMinutes    int
	TargetMinutes    int
	WorkedDays       int
	LateCount        int
	EarlyLeaveCount  int
	MissingCheckOuts int
	MissedWorkdays   int
	SuspiciousCount  int
}

func (m *ReportMailer) reportEmployeeRows(
	ctx context.Context,
	overview *domain.AdminEmployeesMonthOverview,
	suspicious *domain.AdminSuspiciousActivity,
	from time.Time,
	to time.Time,
) ([]adminReportEmployeeRow, error) {
	if overview == nil {
		return nil, nil
	}

	suspiciousCounts := reportSuspiciousCountsByUser(suspicious)
	rows := make([]adminReportEmployeeRow, 0, len(overview.Employees))
	for _, employee := range overview.Employees {
		detail, err := m.adminService.EmployeeMonth(ctx, employee.UserId, from, to)
		if err != nil {
			return nil, err
		}
		if detail == nil {
			continue
		}

		targetMinutes := reportWorkdayCount(detail.AttendanceDays) * overview.TargetMinutesPerDay
		rows = append(rows, adminReportEmployeeRow{
			FullName:         detail.FullName,
			Email:            detail.Email,
			HoursText:        fmt.Sprintf("%s / %s", reportMinutesText(detail.WorkedMinutes), reportMinutesText(targetMinutes)),
			WorkedMinutes:    detail.WorkedMinutes,
			TargetMinutes:    targetMinutes,
			WorkedDays:       detail.WorkedDays,
			LateCount:        detail.LateCount,
			EarlyLeaveCount:  detail.EarlyLeaveCount,
			MissingCheckOuts: reportMissingCheckOutCount(detail.AttendanceDays),
			MissedWorkdays:   reportMissedWorkdayCount(detail.AttendanceDays),
			SuspiciousCount:  suspiciousCounts[detail.UserId.String()],
		})
	}

	sort.SliceStable(rows, func(i int, j int) bool {
		return rows[i].FullName < rows[j].FullName
	})

	return rows, nil
}

func reportStats(
	employees *domain.AdminEmployeesMonthOverview,
	suspicious *domain.AdminSuspiciousActivity,
) adminReportStats {
	stats := adminReportStats{}
	if employees != nil {
		stats.Employees = len(employees.Employees)
		for _, employee := range employees.Employees {
			if employee.LateCount > 0 {
				stats.LateEmployees++
			}
			if employee.EarlyLeaveCount > 0 {
				stats.EarlyLeaveEmployees++
			}
		}
	}

	stats.SuspiciousEmployees = len(reportSuspiciousCountsByUser(suspicious))

	return stats
}

func reportSuspiciousCountsByUser(activity *domain.AdminSuspiciousActivity) map[string]int {
	counts := make(map[string]int)
	if activity == nil {
		return counts
	}

	for _, match := range activity.DeviceMatches {
		counts[match.Owner.UserId.String()]++
		counts[match.Event.UserId.String()]++
	}
	for _, match := range activity.IPMatches {
		counts[match.Event.UserId.String()]++
		counts[match.PreviousEvent.UserId.String()]++
	}

	return counts
}

func reportWorkdayCount(days []domain.AttendanceDaySummary) int {
	count := 0
	for _, day := range days {
		if reportIsWorkday(day.Date) {
			count++
		}
	}
	return count
}

func reportMissedWorkdayCount(days []domain.AttendanceDaySummary) int {
	count := 0
	for _, day := range days {
		if reportIsWorkday(day.Date) && day.CheckInAt == nil {
			count++
		}
	}
	return count
}

func reportMissingCheckOutCount(days []domain.AttendanceDaySummary) int {
	count := 0
	for _, day := range days {
		if reportIsWorkday(day.Date) && day.CheckInAt != nil && day.CheckOutAt == nil {
			count++
		}
	}
	return count
}

func reportIsWorkday(date time.Time) bool {
	weekday := date.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

func reportMinutesText(minutes int) string {
	hours := minutes / 60
	rest := minutes % 60
	if rest == 0 {
		return fmt.Sprintf("%d ч", hours)
	}
	return fmt.Sprintf("%d ч %d мин", hours, rest)
}

func activeAdminEmails(access []domain.AdminAccess) []string {
	result := make([]string, 0, len(access))
	for _, item := range access {
		if item.RevokedAt == nil {
			result = append(result, item.Email)
		}
	}
	return result
}
