create extension if not exists "uuid-ossp";

create table users (
    id uuid primary key default uuid_generate_v4(),
    google_sub text unique,
    email text not null unique,
    password_hash text,
    last_name text not null,
    first_name text not null,
    middle_name text,
    full_name text not null,
    created_at timestamptz not null default now()
);

create table sessions (
    id uuid primary key default uuid_generate_v4(),
    user_id uuid not null references users(id) on delete cascade,
    created_at timestamptz not null default now(),
    expires_at timestamptz not null,
    revoked_at timestamptz,
    constraint sessions_expires_after_created check (expires_at > created_at)
);

create index sessions_user_id_idx on sessions(user_id);
create index sessions_expires_at_idx on sessions(expires_at);

-- business_date is a date according to server business time (from .env)
create table attendance_records (
    id uuid primary key default uuid_generate_v4(),
    user_id uuid not null references users(id) on delete cascade,
    business_date date not null,
    created_at timestamptz not null default now(),

    constraint attendance_records_user_business_date_unique
        unique (user_id, business_date)
);

create index attendance_records_business_date_idx on attendance_records(business_date);
create index attendance_records_user_id_idx on attendance_records(user_id);

create table attendance_events (
    id uuid primary key default uuid_generate_v4(),
    record_id uuid not null references attendance_records(id) on delete cascade,

    event_type text not null
        check (event_type in ('check_in', 'check_out')),

    event_at timestamptz not null default now(),

    status text not null default 'normal'
        check (
            status in (
                'normal',
                'outside_allowed_time',
                'system_outage'
            )
        ),

    phone_model text not null,
    browser text not null,
    device_id text not null,
    external_ip text not null,

    created_at timestamptz not null default now(),

    constraint attendance_events_record_type_unique
        unique (record_id, event_type)
);

create index attendance_events_record_id_idx on attendance_events(record_id);
create index attendance_events_event_at_idx on attendance_events(event_at);
create index attendance_events_event_type_idx on attendance_events(event_type);

-- table for recovery logic after server down
create table system_heartbeat (
    id int primary key default 1, 
    last_seen_at timestamptz not null default now(),
    constraint system_heartbeat_singleton check (id = 1)
);

insert into system_heartbeat (id)
values (1)
on conflict (id) do nothing;

-- tracks system downtime to identify periods with missing or unreliable data
-- started_at: last recorded [system_heartbeat] before failure
-- ended_at: time of successful server restart
-- this table is used to exclude "system fault" periods 
--      from performance metrics and reports
create table system_outages (
    id uuid primary key default uuid_generate_v4(),
    started_at timestamptz not null,
    ended_at timestamptz not null,
    reason text,
    created_at timestamptz not null default now(),
    affected_business_date date,
    impacts_work_hours boolean not null default false,
    resolved_at timestamptz,
    resolved_by text,
    resolution_note text,
    constraint system_outages_ended_after_started check (ended_at >= started_at)
);

create index system_outages_started_at_idx on system_outages(started_at);
create index system_outages_ended_at_idx on system_outages(ended_at);
create index system_outages_affected_business_date_idx on system_outages(affected_business_date);

create table attendance_adjustments (
    id uuid primary key default uuid_generate_v4(),
    user_id uuid not null references users(id) on delete cascade,
    business_date date not null,
    event_type text not null check (event_type in ('check_in', 'check_out')),
    old_event_at timestamptz,
    new_event_at timestamptz,
    reason text not null,
    outage_id uuid references system_outages(id),
    created_by_admin_email text not null,
    created_at timestamptz not null default now()
);

create index attendance_adjustments_user_date_idx on attendance_adjustments(user_id, business_date);
create index attendance_adjustments_outage_id_idx on attendance_adjustments(outage_id);

-- ADMIN PANEL LOGIC --
create table admin_access (
    email text primary key,
    created_at timestamptz not null default now(),
    created_by text null,
    revoked_at timestamptz null
);

create table admin_sessions (
    id uuid primary key default uuid_generate_v4(),
    email text not null references admin_access(email),
    full_name text not null,
    google_sub text not null,
    created_at timestamptz not null default now(),
    expires_at timestamptz not null,
    revoked_at timestamptz,
    constraint admin_sessions_expires_after_created check (expires_at > created_at)
);

create index admin_sessions_email_idx on admin_sessions(email);
create index admin_sessions_expires_at_idx on admin_sessions(expires_at);

create table report_runs (
    id uuid primary key default uuid_generate_v4(),
    period_start date not null,
    period_end date not null,
    sent_at timestamptz not null default now(),
    unique (period_start, period_end)
);
