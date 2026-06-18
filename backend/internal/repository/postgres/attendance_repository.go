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

type AttendanceRepository struct {
	db *sqlx.DB
}

func NewAttendanceRepository(db *sqlx.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

func (r *AttendanceRepository) GetRecordByUserAndDate(
	ctx context.Context,
	userId uuid.UUID,
	businessDate time.Time,
) (*domain.AttendanceRecords, error) {
	const query = `
		select id, user_id, business_date, created_at
		from attendance_records
		where user_id = $1
			and business_date = $2
	`

	q := extractTransaction(ctx, r.db)
	var record domain.AttendanceRecords
	if err := sqlx.GetContext(ctx, q, &record, query, userId, businessDate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &record, nil
}

func (r *AttendanceRepository) CreateOrGetRecord(
	ctx context.Context,
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

	q := extractTransaction(ctx, r.db)
	var record domain.AttendanceRecords
	if err := sqlx.GetContext(ctx, q, &record, query, userId, businessDate); err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *AttendanceRepository) GetEventByRecordAndType(
	ctx context.Context,
	recordId uuid.UUID,
	eventType string,
) (*domain.AttendanceEvents, error) {
	const query = `
		select id, record_id, event_type, event_at, status,
			phone_model, browser, device_id, external_ip, created_at
		from attendance_events
		where record_id = $1
			and event_type = $2
	`

	q := extractTransaction(ctx, r.db)
	var event domain.AttendanceEvents
	if err := sqlx.GetContext(ctx, q, &event, query, recordId, eventType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &event, nil
}

func (r *AttendanceRepository) CreateEvent(
	ctx context.Context,
	input domain.CreateAttendanceEventInput,
) (*domain.AttendanceEvents, error) {
	const query = `
		insert into attendance_events (
			record_id, event_type, status,
			phone_model, browser, device_id, external_ip
		)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id, record_id, event_type, event_at, status,
			phone_model, browser, device_id, external_ip, created_at
	`

	q := extractTransaction(ctx, r.db)
	var event domain.AttendanceEvents
	if err := sqlx.GetContext(
		ctx,
		q,
		&event,
		query,
		input.RecordId,
		input.EventType,
		input.Status,
		input.PhoneModel,
		input.Browser,
		input.DeviceId,
		input.ExternalIp,
	); err != nil {
		return nil, err
	}

	return &event, nil
}

func (r *AttendanceRepository) GetRangeEventRows(
	ctx context.Context,
	userId uuid.UUID,
	from time.Time,
	to time.Time,
) ([]domain.AttendanceRangeEventRow, error) {
	const query = `
		select
			day_rows.business_date,
			ci.event_at as check_in_at,
			co.event_at as check_out_at,
			coalesce(ado.status = 'voided' and ado.restored_at is null, false) as voided,
			ado.reason as void_reason,
			ado.created_by_admin_email as voided_by_admin,
			ado.created_at as voided_at
		from (
			select business_date
			from attendance_records
			where user_id = $1
				and business_date >= $2
				and business_date <= $3
			union
			select business_date
			from attendance_day_overrides
			where user_id = $1
				and business_date >= $2
				and business_date <= $3
				and restored_at is null
		) day_rows
		left join attendance_records ar
			on ar.user_id = $1
			and ar.business_date = day_rows.business_date
		left join attendance_events ci
			on ci.record_id = ar.id
			and ci.event_type = 'check_in'
		left join attendance_events co
			on co.record_id = ar.id
			and co.event_type = 'check_out'
		left join attendance_day_overrides ado
			on ado.user_id = $1
			and ado.business_date = day_rows.business_date
			and ado.restored_at is null
		order by day_rows.business_date
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AttendanceRangeEventRow, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, userId, from, to); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AttendanceRepository) UpsertExplanation(
	ctx context.Context,
	input domain.CreateAttendanceExplanationInput,
) (*domain.AttendanceExplanation, error) {
	const query = `
		insert into attendance_explanations (
			user_id,
			business_date,
			reason_type,
			comment,
			status,
			reviewed_by_admin_email,
			reviewed_at,
			review_note
		)
		values ($1, $2, $3, $4, 'pending', null, null, null)
		on conflict (user_id, business_date, reason_type) do update set
			comment = excluded.comment,
			status = 'pending',
			reviewed_by_admin_email = null,
			reviewed_at = null,
			review_note = null,
			updated_at = now()
		returning id, user_id, business_date, reason_type, comment, status,
			reviewed_by_admin_email, reviewed_at, review_note, created_at, updated_at
	`

	q := extractTransaction(ctx, r.db)
	var row domain.AttendanceExplanation
	if err := sqlx.GetContext(
		ctx,
		q,
		&row,
		query,
		input.UserId,
		input.BusinessDate,
		input.ReasonType,
		input.Comment,
	); err != nil {
		return nil, err
	}

	return &row, nil
}

func (r *AttendanceRepository) ListExplanationsByUserRange(
	ctx context.Context,
	userId uuid.UUID,
	from time.Time,
	to time.Time,
) ([]domain.AttendanceExplanation, error) {
	const query = `
		select id, user_id, business_date, reason_type, comment, status,
			reviewed_by_admin_email, reviewed_at, review_note, created_at, updated_at
		from attendance_explanations
		where user_id = $1
			and business_date >= $2
			and business_date <= $3
		order by business_date, created_at
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AttendanceExplanation, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, userId, from, to); err != nil {
		return nil, err
	}

	return rows, nil
}
