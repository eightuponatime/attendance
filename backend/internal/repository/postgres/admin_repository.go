package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"attendance/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AdminRepository struct {
	db *sqlx.DB
}

func NewAdminRepository(db *sqlx.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) IsActiveAdminByEmail(ctx context.Context, email string) (bool, error) {
	const query = `
		select exists (
			select 1
			from admin_access
			where email = lower($1)
				and revoked_at is null
		)
	`

	q := extractTransaction(ctx, r.db)
	var exists bool
	if err := sqlx.GetContext(ctx, q, &exists, query, email); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *AdminRepository) ListAccess(ctx context.Context) ([]domain.AdminAccess, error) {
	const query = `
		select
			aa.email,
			latest_session.full_name,
			aa.created_at,
			aa.created_by,
			aa.revoked_at
		from admin_access aa
		left join lateral (
			select admin_sessions.full_name
			from admin_sessions
			where admin_sessions.email = aa.email
			order by admin_sessions.created_at desc
			limit 1
		) latest_session on true
		order by aa.revoked_at nulls first, aa.email
	`

	q := extractTransaction(ctx, r.db)
	var rows []domain.AdminAccess
	if err := sqlx.SelectContext(ctx, q, &rows, query); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AdminRepository) UpsertAccess(
	ctx context.Context,
	input domain.CreateAdminAccessInput,
) (*domain.AdminAccess, error) {
	const query = `
		insert into admin_access (email, created_by)
		values ($1, $2)
		on conflict (email) do update
		set revoked_at = null
		returning email, null::text as full_name, created_at, created_by, revoked_at
	`

	q := extractTransaction(ctx, r.db)
	var row domain.AdminAccess
	if err := sqlx.GetContext(ctx, q, &row, query, input.Email, input.CreatedBy); err != nil {
		return nil, err
	}

	return &row, nil
}

func (r *AdminRepository) CreateSession(
	ctx context.Context,
	input domain.CreateAdminSessionInput,
) (*domain.AdminSession, error) {
	const query = `
		insert into admin_sessions (email, full_name, google_sub, expires_at)
		values (lower($1), $2, $3, $4)
		returning id, email, full_name, google_sub, created_at, expires_at, revoked_at
	`

	q := extractTransaction(ctx, r.db)
	var session domain.AdminSession
	if err := sqlx.GetContext(
		ctx,
		q,
		&session,
		query,
		input.Email,
		input.FullName,
		input.GoogleSub,
		input.ExpiresAt,
	); err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *AdminRepository) GetValidSessionByID(ctx context.Context, id uuid.UUID) (*domain.AdminSession, error) {
	const query = `
		select id, email, full_name, google_sub, created_at, expires_at, revoked_at
		from admin_sessions
		where id = $1
			and expires_at > now()
			and revoked_at is null
	`

	q := extractTransaction(ctx, r.db)
	var session domain.AdminSession
	if err := sqlx.GetContext(ctx, q, &session, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &session, nil
}

func (r *AdminRepository) RevokeAccess(ctx context.Context, email string) error {
	const query = `
		update admin_access
		set revoked_at = now()
		where email = $1
			and revoked_at is null
	`

	q := extractTransaction(ctx, r.db)
	_, err := q.ExecContext(ctx, query, email)
	return err
}

func (r *AdminRepository) ListReports(ctx context.Context) ([]domain.AdminReportRun, error) {
	const query = `
		select id, period_start, period_end, sent_at
		from report_runs
		order by period_start desc
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AdminReportRun, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AdminRepository) GetReportRunByPeriod(
	ctx context.Context,
	from time.Time,
	to time.Time,
) (*domain.AdminReportRun, error) {
	const query = `
		select id, period_start, period_end, sent_at
		from report_runs
		where period_start = $1
			and period_end = $2
	`

	q := extractTransaction(ctx, r.db)
	var row domain.AdminReportRun
	if err := sqlx.GetContext(ctx, q, &row, query, from, to); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &row, nil
}

func (r *AdminRepository) CreateReportRun(
	ctx context.Context,
	input domain.CreateAdminReportRunInput,
) (*domain.AdminReportRun, error) {
	const query = `
		insert into report_runs (period_start, period_end)
		values ($1, $2)
		on conflict (period_start, period_end) do update
		set sent_at = report_runs.sent_at
		returning id, period_start, period_end, sent_at
	`

	q := extractTransaction(ctx, r.db)
	var row domain.AdminReportRun
	if err := sqlx.GetContext(ctx, q, &row, query, input.PeriodStart, input.PeriodEnd); err != nil {
		return nil, err
	}

	return &row, nil
}

func (r *AdminRepository) ListSessions(ctx context.Context) ([]domain.AdminSession, error) {
	const query = `
		select s.id, s.email, s.full_name, s.google_sub, s.created_at, s.expires_at, s.revoked_at
		from admin_sessions s
		join admin_access aa on aa.email = lower(s.email)
		order by s.revoked_at nulls first, s.created_at desc
	`

	q := extractTransaction(ctx, r.db)
	var rows []domain.AdminSession
	if err := sqlx.SelectContext(ctx, q, &rows, query); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AdminRepository) RevokeSession(ctx context.Context, sessionId uuid.UUID) error {
	const query = `
		update admin_sessions
		set revoked_at = now()
		where id = $1
			and revoked_at is null
	`

	q := extractTransaction(ctx, r.db)
	_, err := q.ExecContext(ctx, query, sessionId)
	return err
}

func (r *AdminRepository) RevokeSessionsByEmail(ctx context.Context, email string) error {
	const query = `
		update admin_sessions s
		set revoked_at = now()
		where lower(s.email) = $1
			and s.revoked_at is null
	`

	q := extractTransaction(ctx, r.db)
	_, err := q.ExecContext(ctx, query, email)
	return err
}

func (r *AdminRepository) ListEmployeeMonthRows(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]domain.AdminEmployeeMonthRow, error) {
	const query = `
		with employee_dates as (
			select ar.user_id, ar.business_date
			from attendance_records ar
			where ar.business_date >= $1
				and ar.business_date <= $2
			union
			select ado.user_id, ado.business_date
			from attendance_day_overrides ado
			where ado.business_date >= $1
				and ado.business_date <= $2
				and ado.restored_at is null
		)
		select
			u.id as user_id,
			u.email,
			u.full_name,
			ed.business_date,
			ci.event_at as check_in_at,
			co.event_at as check_out_at,
			coalesce(ado.status = 'voided' and ado.restored_at is null, false) as voided,
			ado.reason as void_reason,
			ado.created_by_admin_email as voided_by_admin,
			ado.created_at as voided_at
		from users u
		left join employee_dates ed
			on ed.user_id = u.id
		left join attendance_records ar
			on ar.user_id = u.id
			and ar.business_date = ed.business_date
		left join attendance_day_overrides ado
			on ado.user_id = u.id
			and ado.business_date = ed.business_date
			and ado.restored_at is null
		left join attendance_events ci
			on ci.record_id = ar.id
			and ci.event_type = 'check_in'
		left join attendance_events co
			on co.record_id = ar.id
			and co.event_type = 'check_out'
		order by u.full_name, u.email, ed.business_date
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AdminEmployeeMonthRow, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, from, to); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AdminRepository) ListEmployeeMonthRowsByUser(
	ctx context.Context,
	userId uuid.UUID,
	from time.Time,
	to time.Time,
) ([]domain.AdminEmployeeMonthRow, error) {
	const query = `
		with employee_dates as (
			select ar.user_id, ar.business_date
			from attendance_records ar
			where ar.user_id = $1
				and ar.business_date >= $2
				and ar.business_date <= $3
			union
			select ado.user_id, ado.business_date
			from attendance_day_overrides ado
			where ado.user_id = $1
				and ado.business_date >= $2
				and ado.business_date <= $3
				and ado.restored_at is null
		)
		select
			u.id as user_id,
			u.email,
			u.full_name,
			ed.business_date,
			ci.event_at as check_in_at,
			co.event_at as check_out_at,
			coalesce(ado.status = 'voided' and ado.restored_at is null, false) as voided,
			ado.reason as void_reason,
			ado.created_by_admin_email as voided_by_admin,
			ado.created_at as voided_at
		from users u
		left join employee_dates ed
			on ed.user_id = u.id
		left join attendance_records ar
			on ar.user_id = u.id
			and ar.business_date = ed.business_date
		left join attendance_day_overrides ado
			on ado.user_id = u.id
			and ado.business_date = ed.business_date
			and ado.restored_at is null
		left join attendance_events ci
			on ci.record_id = ar.id
			and ci.event_type = 'check_in'
		left join attendance_events co
			on co.record_id = ar.id
			and co.event_type = 'check_out'
		where u.id = $1
		order by ed.business_date
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AdminEmployeeMonthRow, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, userId, from, to); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AdminRepository) ListAttendanceEvents(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]domain.AdminAttendanceEventRow, error) {
	const query = `
		select
			ae.id as event_id,
			u.id as user_id,
			u.email,
			u.full_name,
			ar.business_date,
			ae.event_type,
			ae.event_at,
			ae.device_id,
			ae.external_ip
		from attendance_events ae
		join attendance_records ar on ar.id = ae.record_id
		join users u on u.id = ar.user_id
		left join attendance_day_overrides ado
			on ado.user_id = ar.user_id
			and ado.business_date = ar.business_date
			and ado.restored_at is null
		where ar.business_date >= $1
			and ar.business_date <= $2
			and ae.status <> 'system_outage'
			and ae.device_id <> 'admin-adjustment'
			and ae.external_ip <> 'internal'
			and ado.user_id is null
		order by ae.event_at
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AdminAttendanceEventRow, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, from, to); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AdminRepository) GetSystemHeartbeat(ctx context.Context) (*domain.SystemHeartbeat, error) {
	const query = `
		select id, last_seen_at
		from system_heartbeat
		where id = 1
	`

	q := extractTransaction(ctx, r.db)
	var row domain.SystemHeartbeat
	if err := sqlx.GetContext(ctx, q, &row, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &row, nil
}

func (r *AdminRepository) UpdateSystemHeartbeat(ctx context.Context, seenAt time.Time) error {
	const query = `
		insert into system_heartbeat (id, last_seen_at)
		values (1, $1)
		on conflict (id) do update set last_seen_at = excluded.last_seen_at
	`

	q := extractTransaction(ctx, r.db)
	_, err := q.ExecContext(ctx, query, seenAt)
	return err
}

func (r *AdminRepository) CreateSystemOutage(
	ctx context.Context,
	input domain.CreateSystemOutageInput,
) (*domain.SystemOutage, error) {
	const query = `
		insert into system_outages (
			started_at,
			ended_at,
			reason,
			affected_business_date,
			impacts_work_hours
		)
		values ($1, $2, $3, $4, $5)
		returning id, started_at, ended_at, reason, created_at, affected_business_date,
			impacts_work_hours, resolved_at, resolved_by, resolution_note
	`

	q := extractTransaction(ctx, r.db)
	var row domain.SystemOutage
	if err := sqlx.GetContext(
		ctx,
		q,
		&row,
		query,
		input.StartedAt,
		input.EndedAt,
		input.Reason,
		input.AffectedBusinessDate,
		input.ImpactsWorkHours,
	); err != nil {
		return nil, err
	}

	return &row, nil
}

func (r *AdminRepository) ListSystemOutages(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]domain.SystemOutage, error) {
	const query = `
		select id, started_at, ended_at, reason, created_at, affected_business_date,
			impacts_work_hours, resolved_at, resolved_by, resolution_note
		from system_outages
		where impacts_work_hours = true
			and (
				(affected_business_date is not null and affected_business_date >= $1 and affected_business_date <= $2)
				or (started_at < ($2::timestamptz + interval '1 day') and ended_at >= $1::timestamptz)
			)
		order by started_at desc
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.SystemOutage, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, from, to); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AdminRepository) GetSystemOutageByID(ctx context.Context, id uuid.UUID) (*domain.SystemOutage, error) {
	const query = `
		select id, started_at, ended_at, reason, created_at, affected_business_date,
			impacts_work_hours, resolved_at, resolved_by, resolution_note
		from system_outages
		where id = $1
	`

	q := extractTransaction(ctx, r.db)
	var row domain.SystemOutage
	if err := sqlx.GetContext(ctx, q, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &row, nil
}

func (r *AdminRepository) ListOutageDayEmployees(
	ctx context.Context,
	businessDate time.Time,
) ([]domain.AdminOutageDayEmployeeRow, error) {
	const query = `
		select
			u.id as user_id,
			u.email,
			u.full_name,
			ci.event_at as check_in_at,
			co.event_at as check_out_at
		from users u
		left join attendance_records ar
			on ar.user_id = u.id
			and ar.business_date = $1
		left join attendance_events ci
			on ci.record_id = ar.id
			and ci.event_type = 'check_in'
		left join attendance_events co
			on co.record_id = ar.id
			and co.event_type = 'check_out'
		order by u.full_name, u.email
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AdminOutageDayEmployeeRow, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, businessDate); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AdminRepository) UpsertAttendanceEventAt(
	ctx context.Context,
	input domain.UpsertAttendanceEventAtInput,
) (*domain.UpsertAttendanceEventAtResult, error) {
	q := extractTransaction(ctx, r.db)
	record, err := r.createOrGetAttendanceRecord(ctx, q, input.UserId, input.BusinessDate)
	if err != nil {
		return nil, err
	}

	oldEventAt, err := r.getAttendanceEventTime(ctx, q, record.Id, input.EventType)
	if err != nil {
		return nil, err
	}

	const eventQuery = `
		insert into attendance_events (
			record_id, event_type, event_at, status,
			phone_model, browser, device_id, external_ip
		)
		values ($1, $2, $3, $4, 'Admin adjustment', 'Admin panel', 'admin-adjustment', 'internal')
		on conflict (record_id, event_type) do update set
			event_at = excluded.event_at,
			status = excluded.status,
			phone_model = excluded.phone_model,
			browser = excluded.browser,
			device_id = excluded.device_id,
			external_ip = excluded.external_ip
	`
	status := input.Status
	if status == "" {
		status = "normal"
	}
	if _, err := q.ExecContext(ctx, eventQuery, record.Id, input.EventType, input.EventAt, status); err != nil {
		return nil, err
	}

	return &domain.UpsertAttendanceEventAtResult{
		OldEventAt: oldEventAt,
		NewEventAt: input.EventAt,
	}, nil
}

func (r *AdminRepository) SetAttendanceEventAt(
	ctx context.Context,
	input domain.SetAttendanceEventAtInput,
) error {
	q := extractTransaction(ctx, r.db)
	if input.EventAt == nil {
		const deleteQuery = `
			delete from attendance_events ae
			using attendance_records ar
			where ae.record_id = ar.id
				and ar.user_id = $1
				and ar.business_date = $2
				and ae.event_type = $3
		`
		_, err := q.ExecContext(ctx, deleteQuery, input.UserId, input.BusinessDate, input.EventType)
		return err
	}

	_, err := r.UpsertAttendanceEventAt(ctx, domain.UpsertAttendanceEventAtInput{
		UserId:       input.UserId,
		BusinessDate: input.BusinessDate,
		EventType:    input.EventType,
		EventAt:      *input.EventAt,
		Status:       input.Status,
	})
	return err
}

func (r *AdminRepository) CreateAttendanceAdjustment(
	ctx context.Context,
	input domain.CreateAttendanceAdjustmentInput,
) error {
	const query = `
		insert into attendance_adjustments (
			user_id,
			business_date,
			event_type,
			old_event_at,
			new_event_at,
			reason,
			outage_id,
			created_by_admin_email
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	q := extractTransaction(ctx, r.db)
	var outageId any
	if input.OutageId != uuid.Nil {
		outageId = input.OutageId
	}

	if _, err := q.ExecContext(
		ctx,
		query,
		input.UserId,
		input.BusinessDate,
		input.EventType,
		input.OldEventAt,
		input.NewEventAt,
		input.Reason,
		outageId,
		input.CreatedByAdminEmail,
	); err != nil {
		return err
	}

	action := "check_in_changed"
	if input.EventType == "check_out" {
		action = "check_out_changed"
	}
	auditInput := domain.AdminAuditInput{
		AdminEmail:     input.CreatedByAdminEmail,
		UserId:         input.UserId,
		BusinessDate:   input.BusinessDate,
		Action:         action,
		DecisionSource: input.DecisionSource,
		Reason:         input.Reason,
	}
	if input.EventType == "check_out" {
		auditInput.OldCheckOutAt = input.OldEventAt
		auditInput.NewCheckOutAt = &input.NewEventAt
	} else {
		auditInput.OldCheckInAt = input.OldEventAt
		auditInput.NewCheckInAt = &input.NewEventAt
	}
	auditInput.ExplanationId = input.ExplanationId

	return r.createAdminAuditLog(ctx, q, auditInput)
}

func (r *AdminRepository) VoidAttendanceDay(
	ctx context.Context,
	input domain.AdminVoidDayInput,
	oldCheckInAt *time.Time,
	oldCheckOutAt *time.Time,
) error {
	q := extractTransaction(ctx, r.db)
	const overrideQuery = `
		insert into attendance_day_overrides (
			user_id, business_date, status, reason, created_by_admin_email,
			restored_by_admin_email, restored_at, restore_reason
		)
		values ($1, $2, 'voided', $3, $4, null, null, null)
		on conflict (user_id, business_date) do update set
			status = 'voided',
			reason = excluded.reason,
			created_by_admin_email = excluded.created_by_admin_email,
			created_at = now(),
			restored_by_admin_email = null,
			restored_at = null,
			restore_reason = null
	`
	if _, err := q.ExecContext(ctx, overrideQuery, input.UserId, input.BusinessDate, input.Reason, input.AdminEmail); err != nil {
		return err
	}

	return r.createAdminAuditLog(ctx, q, domain.AdminAuditInput{
		AdminEmail:     input.AdminEmail,
		UserId:         input.UserId,
		ExplanationId:  input.ExplanationId,
		BusinessDate:   input.BusinessDate,
		Action:         "day_voided",
		OldCheckInAt:   oldCheckInAt,
		OldCheckOutAt:  oldCheckOutAt,
		DecisionSource: input.DecisionSource,
		Reason:         input.Reason,
	})
}

func (r *AdminRepository) RestoreAttendanceDay(
	ctx context.Context,
	input domain.AdminVoidDayInput,
	oldCheckInAt *time.Time,
	oldCheckOutAt *time.Time,
) error {
	q := extractTransaction(ctx, r.db)
	const overrideQuery = `
		update attendance_day_overrides
		set restored_by_admin_email = $3,
			restored_at = now(),
			restore_reason = $4
		where user_id = $1
			and business_date = $2
			and restored_at is null
	`
	if _, err := q.ExecContext(ctx, overrideQuery, input.UserId, input.BusinessDate, input.AdminEmail, input.Reason); err != nil {
		return err
	}

	return r.createAdminAuditLog(ctx, q, domain.AdminAuditInput{
		AdminEmail:     input.AdminEmail,
		UserId:         input.UserId,
		ExplanationId:  input.ExplanationId,
		BusinessDate:   input.BusinessDate,
		Action:         "day_restored",
		NewCheckInAt:   oldCheckInAt,
		NewCheckOutAt:  oldCheckOutAt,
		DecisionSource: input.DecisionSource,
		Reason:         input.Reason,
	})
}

func (r *AdminRepository) createAdminAuditLog(ctx context.Context, q sqlx.ExtContext, input domain.AdminAuditInput) error {
	const query = `
		insert into admin_audit_logs (
			admin_email, user_id, explanation_id, business_date, action,
			old_check_in_at, old_check_out_at, new_check_in_at, new_check_out_at,
			decision_source, reason
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	decisionSource := input.DecisionSource
	if decisionSource == "" {
		decisionSource = "admin_decision"
	}
	var userId any
	if input.UserId != uuid.Nil {
		userId = input.UserId
	}
	var explanationId any
	if input.ExplanationId != uuid.Nil {
		explanationId = input.ExplanationId
	}
	var businessDate any
	if !input.BusinessDate.IsZero() {
		businessDate = input.BusinessDate
	}
	_, err := q.ExecContext(
		ctx,
		query,
		input.AdminEmail,
		userId,
		explanationId,
		businessDate,
		input.Action,
		input.OldCheckInAt,
		input.OldCheckOutAt,
		input.NewCheckInAt,
		input.NewCheckOutAt,
		decisionSource,
		input.Reason,
	)
	return err
}

func (r *AdminRepository) CreateAdminAuditLog(ctx context.Context, input domain.AdminAuditInput) error {
	q := extractTransaction(ctx, r.db)
	return r.createAdminAuditLog(ctx, q, input)
}

func (r *AdminRepository) ListAuditLogs(ctx context.Context, from time.Time, to time.Time) ([]domain.AdminAuditLog, error) {
	const query = `
		select
			l.id,
			l.admin_email,
			l.user_id,
			l.explanation_id,
			u.email,
			u.full_name,
			l.business_date,
			l.action,
			l.old_check_in_at,
			l.old_check_out_at,
			l.new_check_in_at,
			l.new_check_out_at,
			l.decision_source,
			l.reason,
			l.created_at
		from admin_audit_logs l
		left join users u on u.id = l.user_id
		where (
				l.business_date is not null
				and l.business_date >= $1::date
				and l.business_date <= $2::date
			)
			or (
				l.business_date is null
				and l.created_at >= $1
				and l.created_at < ($2::timestamptz + interval '1 day')
			)
		order by l.created_at desc
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AdminAuditLog, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, from, to); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AdminRepository) ListAuditLogsByExplanation(ctx context.Context, explanationId uuid.UUID) ([]domain.AdminAuditLog, error) {
	const query = `
		select
			l.id,
			l.admin_email,
			l.user_id,
			l.explanation_id,
			u.email,
			u.full_name,
			l.business_date,
			l.action,
			l.old_check_in_at,
			l.old_check_out_at,
			l.new_check_in_at,
			l.new_check_out_at,
			l.decision_source,
			l.reason,
			l.created_at
		from admin_audit_logs l
		left join users u on u.id = l.user_id
		where l.explanation_id = $1
		order by l.created_at
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AdminAuditLog, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, explanationId); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AdminRepository) ResolveSystemOutage(
	ctx context.Context,
	outageId uuid.UUID,
	adminEmail string,
	note string,
) error {
	const query = `
		update system_outages
		set resolved_at = now(),
			resolved_by = $2,
			resolution_note = $3
		where id = $1
		returning affected_business_date
	`

	q := extractTransaction(ctx, r.db)
	var affectedBusinessDate sql.NullTime
	if err := sqlx.GetContext(ctx, q, &affectedBusinessDate, query, outageId, adminEmail, note); err != nil {
		return err
	}

	var businessDate time.Time
	if affectedBusinessDate.Valid {
		businessDate = affectedBusinessDate.Time
	}
	return r.createAdminAuditLog(ctx, q, domain.AdminAuditInput{
		AdminEmail:     adminEmail,
		BusinessDate:   businessDate,
		Action:         "system_outage_resolved",
		DecisionSource: "admin_decision",
		Reason:         "system_outage_resolved",
	})
}

func (r *AdminRepository) ListExplanations(
	ctx context.Context,
	from time.Time,
	to time.Time,
	status string,
) ([]domain.AdminExplanationRow, error) {
	query := `
		select
			ae.id,
			ae.user_id,
			u.email,
			u.full_name,
			ae.business_date,
			ae.reason_type,
			ae.comment,
			ae.status,
			ae.reviewed_by_admin_email,
			ae.reviewed_at,
			ae.review_note,
			ae.created_at,
			ae.updated_at,
			ci.event_at as check_in_at,
			co.event_at as check_out_at
		from attendance_explanations ae
		join users u on u.id = ae.user_id
		left join attendance_records ar
			on ar.user_id = ae.user_id
			and ar.business_date = ae.business_date
		left join attendance_events ci
			on ci.record_id = ar.id
			and ci.event_type = 'check_in'
		left join attendance_events co
			on co.record_id = ar.id
			and co.event_type = 'check_out'
		where ae.business_date >= $1
			and ae.business_date <= $2
	`
	args := []any{from, to}
	if status != "" {
		query += " and ae.status = $3"
		args = append(args, status)
	}
	query += " order by ae.status = 'pending' desc, ae.updated_at desc"

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AdminExplanationRow, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, args...); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AdminRepository) GetExplanationByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.AdminExplanationRow, error) {
	const query = `
		select
			ae.id,
			ae.user_id,
			u.email,
			u.full_name,
			ae.business_date,
			ae.reason_type,
			ae.comment,
			ae.status,
			ae.reviewed_by_admin_email,
			ae.reviewed_at,
			ae.review_note,
			ae.created_at,
			ae.updated_at,
			ci.event_at as check_in_at,
			co.event_at as check_out_at
		from attendance_explanations ae
		join users u on u.id = ae.user_id
		left join attendance_records ar
			on ar.user_id = ae.user_id
			and ar.business_date = ae.business_date
		left join attendance_events ci
			on ci.record_id = ar.id
			and ci.event_type = 'check_in'
		left join attendance_events co
			on co.record_id = ar.id
			and co.event_type = 'check_out'
		where ae.id = $1
	`

	q := extractTransaction(ctx, r.db)
	var row domain.AdminExplanationRow
	if err := sqlx.GetContext(ctx, q, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &row, nil
}

func (r *AdminRepository) UpdateExplanationReview(
	ctx context.Context,
	id uuid.UUID,
	status string,
	adminEmail string,
	note string,
) (*domain.AttendanceExplanation, error) {
	const query = `
		update attendance_explanations
		set status = $2,
			reviewed_by_admin_email = $3,
			reviewed_at = now(),
			review_note = nullif($4, ''),
			updated_at = now()
		where id = $1
		returning id, user_id, business_date, reason_type, comment, status,
			reviewed_by_admin_email, reviewed_at, review_note, created_at, updated_at
	`

	q := extractTransaction(ctx, r.db)
	var row domain.AttendanceExplanation
	if err := sqlx.GetContext(ctx, q, &row, query, id, status, adminEmail, note); err != nil {
		return nil, err
	}

	return &row, nil
}

func (r *AdminRepository) ResetExplanationReview(
	ctx context.Context,
	id uuid.UUID,
) (*domain.AttendanceExplanation, error) {
	const query = `
		update attendance_explanations
		set status = 'pending',
			reviewed_by_admin_email = null,
			reviewed_at = null,
			review_note = null,
			updated_at = now()
		where id = $1
		returning id, user_id, business_date, reason_type, comment, status,
			reviewed_by_admin_email, reviewed_at, review_note, created_at, updated_at
	`

	q := extractTransaction(ctx, r.db)
	var row domain.AttendanceExplanation
	if err := sqlx.GetContext(ctx, q, &row, query, id); err != nil {
		return nil, err
	}

	return &row, nil
}

func (r *AdminRepository) createOrGetAttendanceRecord(
	ctx context.Context,
	q sqlx.ExtContext,
	userId uuid.UUID,
	businessDate time.Time,
) (*domain.AttendanceRecords, error) {
	const query = `
		insert into attendance_records (user_id, business_date)
		values ($1, $2)
		on conflict (user_id, business_date) do update set
			business_date = excluded.business_date
		returning id, user_id, business_date, created_at
	`

	var record domain.AttendanceRecords
	if err := sqlx.GetContext(ctx, q, &record, query, userId, businessDate); err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *AdminRepository) getAttendanceEventTime(
	ctx context.Context,
	q sqlx.ExtContext,
	recordId uuid.UUID,
	eventType string,
) (*time.Time, error) {
	const query = `
		select event_at
		from attendance_events
		where record_id = $1
			and event_type = $2
	`

	var eventAt time.Time
	if err := sqlx.GetContext(ctx, q, &eventAt, query, recordId, eventType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &eventAt, nil
}
