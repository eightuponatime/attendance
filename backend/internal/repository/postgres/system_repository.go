package postgres

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type SystemRepository struct {
	db *sqlx.DB
}

func NewSystemRepository(db *sqlx.DB) *SystemRepository {
	return &SystemRepository{db: db}
}

func (r *SystemRepository) ListImpactedBusinessDates(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]time.Time, error) {
	const query = `
		select distinct affected_business_date
		from system_outages
		where impacts_work_hours = true
			and affected_business_date is not null
			and affected_business_date >= $1
			and affected_business_date <= $2
		order by affected_business_date
	`

	q := extractTransaction(ctx, r.db)
	rows := make([]time.Time, 0)
	if err := sqlx.SelectContext(ctx, q, &rows, query, from, to); err != nil {
		return nil, err
	}

	return rows, nil
}
