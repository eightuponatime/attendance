create table attendance_explanations (
    id uuid primary key default uuid_generate_v4(),
    user_id uuid not null references users(id) on delete cascade,
    business_date date not null,
    reason_type text not null check (
        reason_type in (
            'late',
            'early_leave',
            'missing_check_in',
            'missing_check_out',
            'missing_day'
        )
    ),
    comment text not null,
    status text not null default 'pending' check (
        status in ('pending', 'approved', 'rejected')
    ),
    reviewed_by_admin_email text,
    reviewed_at timestamptz,
    review_note text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    constraint attendance_explanations_unique_reason
        unique (user_id, business_date, reason_type)
);

create index attendance_explanations_user_date_idx
    on attendance_explanations(user_id, business_date);

create index attendance_explanations_status_date_idx
    on attendance_explanations(status, business_date);

create index attendance_explanations_business_date_idx
    on attendance_explanations(business_date);
