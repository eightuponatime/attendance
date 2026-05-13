create extension if not exists "uuid-ossp";

-- google sub is a unique if from google workspace
create table users (
    id uuid primary key default uuid_generate_v4(),
    google_sub text not null,
    email text not null,
    full_name text not null,
    created_at timestampz not null default now()
)

create table sessions (
    id uuid primary key default uuid_generate_v4(),
    user_id uuid not null references users(id),
    created_at timestampz not null,
    expires_at timestampz not null
)

-- business_date is a date according to server business time (from .env)
create table attendance_records (
    id uuid primary key default uuid_generate_v4(),
    user_id uuid not null references users(id),
    business_date date not null,
    check_in_at timestampz not null default now(),
    check_out_at timestampz,
    created_at timestampz not null default now()
)

-- table for recovery logic after server down
create table system_heartbeat(
    id uuid primary key default uuid_generate_v4(),
    last_seen_at timestampz not null
)

-- tracks system downtime to identify periods with missing or unreliable data
-- started_at: last recorded [system_heartbeat] before failure
-- ended_at: time of successful server restart
-- this table is used to exclude "system fault" periods 
--      from performance metrics and reports
create table system_outages(
    id uuid primary key default uuid_generate_v4(),
    started_at timestampz not null,
    ended_at timestampz not null,
    reason text,
    created_at timestampz not null
)