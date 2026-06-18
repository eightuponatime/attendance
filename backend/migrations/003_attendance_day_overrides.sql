create table if not exists attendance_day_overrides (
    user_id uuid not null references users(id) on delete cascade,
    business_date date not null,
    status text not null default 'voided',
    reason text not null default '',
    created_by_admin_email text not null default '',
    created_at timestamptz not null default now(),
    restored_by_admin_email text,
    restored_at timestamptz,
    restore_reason text,

    primary key (user_id, business_date)
);

alter table attendance_day_overrides
    add column if not exists status text not null default 'voided',
    add column if not exists reason text not null default '',
    add column if not exists created_by_admin_email text not null default '',
    add column if not exists created_at timestamptz not null default now(),
    add column if not exists restored_by_admin_email text,
    add column if not exists restored_at timestamptz,
    add column if not exists restore_reason text;

alter table attendance_day_overrides
    alter column status drop default,
    alter column reason drop default,
    alter column created_by_admin_email drop default;

do $$
begin
    if not exists (
        select 1
        from pg_constraint
        where conname = 'attendance_day_overrides_status_check'
    ) then
        alter table attendance_day_overrides
            add constraint attendance_day_overrides_status_check
            check (status in ('voided'));
    end if;
end $$;

create index if not exists attendance_day_overrides_business_date_idx
    on attendance_day_overrides(business_date);

create index if not exists attendance_day_overrides_status_date_idx
    on attendance_day_overrides(status, business_date);

create table if not exists admin_audit_logs (
    id uuid primary key default uuid_generate_v4(),
    admin_email text not null,
    user_id uuid references users(id) on delete set null,
    explanation_id uuid references attendance_explanations(id) on delete set null,
    business_date date,
    action text not null,
    old_check_in_at timestamptz,
    old_check_out_at timestamptz,
    new_check_in_at timestamptz,
    new_check_out_at timestamptz,
    decision_source text not null default 'admin_decision',
    reason text not null,
    created_at timestamptz not null default now()
);

alter table admin_audit_logs
    add column if not exists admin_email text not null default '',
    add column if not exists user_id uuid references users(id) on delete set null,
    add column if not exists explanation_id uuid references attendance_explanations(id) on delete set null,
    add column if not exists business_date date,
    add column if not exists action text not null default 'day_voided',
    add column if not exists old_check_in_at timestamptz,
    add column if not exists old_check_out_at timestamptz,
    add column if not exists new_check_in_at timestamptz,
    add column if not exists new_check_out_at timestamptz,
    add column if not exists decision_source text not null default 'admin_decision',
    add column if not exists reason text not null default '',
    add column if not exists created_at timestamptz not null default now();

alter table admin_audit_logs
    alter column admin_email drop default,
    alter column action drop default,
    alter column decision_source drop default,
    alter column reason drop default;

do $$
begin
    if not exists (
        select 1
        from pg_constraint
        where conname = 'admin_audit_logs_action_check'
    ) then
        alter table admin_audit_logs
            add constraint admin_audit_logs_action_check
            check (action in (
                'day_voided',
                'day_restored',
                'check_in_changed',
                'check_out_changed',
                'explanation_approved',
                'explanation_rejected',
                'explanation_rollback',
                'system_outage_resolved'
            ));
    else
        alter table admin_audit_logs
            drop constraint admin_audit_logs_action_check;

        alter table admin_audit_logs
            add constraint admin_audit_logs_action_check
            check (action in (
                'day_voided',
                'day_restored',
                'check_in_changed',
                'check_out_changed',
                'explanation_approved',
                'explanation_rejected',
                'explanation_rollback',
                'system_outage_resolved'
            ));
    end if;
end $$;

do $$
begin
    if not exists (
        select 1
        from pg_constraint
        where conname = 'admin_audit_logs_decision_source_check'
    ) then
        alter table admin_audit_logs
            add constraint admin_audit_logs_decision_source_check
            check (decision_source in (
                'admin_decision',
                'employee_request'
            ));
    end if;
end $$;

create index if not exists admin_audit_logs_created_at_idx
    on admin_audit_logs(created_at desc);

create index if not exists admin_audit_logs_user_date_idx
    on admin_audit_logs(user_id, business_date);

create index if not exists admin_audit_logs_explanation_id_idx
    on admin_audit_logs(explanation_id);

create index if not exists admin_audit_logs_admin_email_idx
    on admin_audit_logs(admin_email);

create index if not exists admin_audit_logs_decision_source_idx
    on admin_audit_logs(decision_source);

do $$
begin
    if exists (
        select 1
        from pg_constraint
        where conname = 'attendance_explanations_reason_type_check'
    ) then
        alter table attendance_explanations
            drop constraint attendance_explanations_reason_type_check;
    end if;

    alter table attendance_explanations
        add constraint attendance_explanations_reason_type_check
        check (reason_type in (
            'late',
            'early_leave',
            'missing_check_in',
            'missing_check_out',
            'missing_day',
            'void_day_request'
        ));
end $$;
