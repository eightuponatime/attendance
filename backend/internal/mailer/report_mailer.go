package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"net"
	"net/smtp"
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
	logger       *zap.SugaredLogger
}

func NewReportMailer(
	cfg *config.Config,
	adminService service.AdminService,
	logger *zap.SugaredLogger,
) *ReportMailer {
	return &ReportMailer{
		cfg:          cfg,
		adminService: adminService,
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

	from, to, periodTitle, due, err := m.latestDueMonthlyRange()
	if err != nil {
		return err
	}
	if !due {
		m.logger.Info("monthly report email skipped: no closed period is due yet")
		return nil
	}

	reportRun, err := m.adminService.GetReportRunByPeriod(ctx, from, to)
	if err != nil {
		return err
	}
	if reportRun != nil {
		m.logger.Infow("monthly report email skipped: period already sent", "period", periodTitle)
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
	displayPeriod := reportPeriodTitle(from)
	subject := fmt.Sprintf("HR отчет за %s", displayPeriod)
	body := m.reportHTML(displayPeriod, stats)

	for _, recipient := range recipients {
		if err := m.send(recipient, subject, body); err != nil {
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

func (m *ReportMailer) reportHTML(periodTitle string, stats adminReportStats) string {
	adminURL := strings.TrimRight(m.cfg.AdminFrontendURLValue(), "/") + "/admin"
	return fmt.Sprintf(`<!doctype html>
<html>
<body style="margin:0;padding:0;background:#f3f6fa;font-family:Arial,sans-serif;color:#071026;">
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#f3f6fa;padding:28px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:640px;border-collapse:separate;border-spacing:0;">
          <tr>
            <td style="padding:28px 30px;border-radius:14px;background:#113650;color:#ffffff;">
              <div style="margin:0 0 12px;color:#c6d4df;font-size:12px;font-weight:700;letter-spacing:1.6px;text-transform:uppercase;">Панель HR</div>
              <div style="margin:0 0 10px;font-size:28px;line-height:1.2;font-weight:800;">Сводка за %s</div>
              <div style="margin:0;color:#e5edf3;font-size:14px;line-height:1.45;">Автоматически сформированная сводка по посещаемости сотрудников.</div>
            </td>
          </tr>
          <tr><td style="height:16px;"></td></tr>
          <tr>
            <td style="padding:22px 24px;border:1px solid #dfe7f1;border-radius:14px;background:#ffffff;">
              <div style="margin:0 0 16px;color:#071026;font-size:21px;line-height:1.25;font-weight:800;">Ключевые показатели</div>
              <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:separate;border-spacing:0 8px;">
                %s
                %s
                %s
                %s
              </table>
            </td>
          </tr>
          <tr><td style="height:16px;"></td></tr>
          <tr>
            <td style="padding:22px 24px;border:1px solid #b9dcff;border-radius:14px;background:#eaf6ff;">
              <div style="margin:0 0 10px;color:#07598f;font-size:20px;line-height:1.25;font-weight:800;">Админ-панель</div>
              <div style="margin:0 0 18px;color:#2e5d7b;font-size:13px;line-height:1.45;">Перейдите в панель, чтобы посмотреть сотрудников, календарь, сбои сервера и детали отметок.</div>
              <a href="%s" style="display:inline-block;padding:13px 18px;border-radius:8px;background:#113650;color:#ffffff;text-decoration:none;font-size:14px;font-weight:800;">Открыть админ-панель</a>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		html.EscapeString(periodTitle),
		statRow("Сотрудников", stats.Employees),
		statRow("С опозданиями", stats.LateEmployees),
		statRow("С ранними уходами", stats.EarlyLeaveEmployees),
		statRow("С подозрительной активностью", stats.SuspiciousEmployees),
		html.EscapeString(adminURL),
	)
}

func statRow(label string, value int) string {
	return fmt.Sprintf(
		`<tr>
                  <td style="padding:13px 16px;border-radius:8px 0 0 8px;background:#f6f8fb;color:#697083;font-size:14px;">%s</td>
                  <td align="right" style="padding:13px 16px;border-radius:0 8px 8px 0;background:#f6f8fb;color:#071026;font-size:22px;font-weight:800;">%d</td>
                </tr>`,
		html.EscapeString(label),
		value,
	)
}

func reportPeriodTitle(from time.Time) string {
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
	return fmt.Sprintf("%s %d", months[int(from.Month())-1], from.Year())
}

func (m *ReportMailer) send(to string, subject string, htmlBody string) error {
	headers := map[string]string{
		"From":         m.cfg.SMTPFrom,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": `text/html; charset="UTF-8"`,
	}

	var message strings.Builder
	for key, value := range headers {
		message.WriteString(key)
		message.WriteString(": ")
		message.WriteString(value)
		message.WriteString("\r\n")
	}
	message.WriteString("\r\n")
	message.WriteString(htmlBody)

	addr := net.JoinHostPort(m.cfg.SMTPHost, m.cfg.SMTPPort)
	if m.cfg.SMTPPort == "465" {
		return m.sendTLS(addr, to, []byte(message.String()))
	}

	return m.sendStartTLS(addr, to, []byte(message.String()))
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

func (m *ReportMailer) latestDueMonthlyRange() (time.Time, time.Time, string, bool, error) {
	location, err := time.LoadLocation(m.cfg.BusinessTimezone)
	if err != nil {
		return time.Time{}, time.Time{}, "", false, err
	}

	now := time.Now().In(location)
	clock, err := time.Parse("15:04", m.cfg.ReportEmailTime)
	if err != nil {
		return time.Time{}, time.Time{}, "", false, err
	}

	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
	dueAt := time.Date(now.Year(), now.Month(), 1, clock.Hour(), clock.Minute(), 0, 0, location)
	if now.Before(dueAt) {
		return time.Time{}, time.Time{}, "", false, nil
	}

	from := currentMonthStart.AddDate(0, -1, 0)
	to := currentMonthStart.AddDate(0, 0, -1)
	title := from.Format("2006-01")
	return from, to, title, true, nil
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

	now := time.Now().In(location)
	next := time.Date(now.Year(), now.Month(), 1, clock.Hour(), clock.Minute(), 0, 0, location)
	if !next.After(now) {
		next = next.AddDate(0, 1, 0)
	}

	return time.Until(next), nil
}

type adminReportStats struct {
	Employees           int
	LateEmployees       int
	EarlyLeaveEmployees int
	SuspiciousEmployees int
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

	users := make(map[string]struct{})
	if suspicious != nil {
		for _, match := range suspicious.DeviceMatches {
			users[match.Event.UserId.String()] = struct{}{}
		}
		for _, match := range suspicious.IPMatches {
			users[match.Event.UserId.String()] = struct{}{}
		}
	}
	stats.SuspiciousEmployees = len(users)

	return stats
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
