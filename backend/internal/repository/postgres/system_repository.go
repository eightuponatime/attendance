package postgres

import (
	"context"
	"time"

	"attendance/internal/domain"

	"github.com/jmoiron/sqlx"
)

type SystemRepository struct {
	db *sqlx.DB
}

func NewSystemRepository(db *sqlx.DB) *SystemRepository {
	return &SystemRepository{db: db}
}

func (r *SystemRepository) ListSystemOutages(
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
