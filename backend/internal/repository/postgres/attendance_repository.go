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
			ar.business_date,
			ci.event_at as check_in_at,
			co.event_at as check_out_at
		from attendance_records ar
		left join attendance_events ci
			on ci.record_id = ar.id
			and ci.event_type = 'check_in'
		left join attendance_events co
			on co.record_id = ar.id
			and co.event_type = 'check_out'
		where ar.user_id = $1
			and ar.business_date >= $2
			and ar.business_date <= $3
		order by ar.business_date
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]domain.AttendanceRangeEventRow, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, userId, from, to); err != nil {
		return nil, err
	}

	return rows, nil
}
