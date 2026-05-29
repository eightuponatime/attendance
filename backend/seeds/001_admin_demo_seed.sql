-- Admin dashboard demo seed for manual execution in DBeaver.
-- Safe to run multiple times.
-- Adds several extra users, April/May 2026 attendance data, one April report run,
-- and deliberate device/IP overlaps for suspicious activity checks.
-- Existing kitchen@goldencompass.kz is not inserted into users here; its known user_id is reused.

with user_seed (id, google_sub, email, full_name, created_at) as (
    values
        (
            '0c450c7f-9cf0-4323-87f0-d3bfbd332a41'::uuid,
            'demo-google-sub-kurban',
            'kurban@goldencompass.kz',
            'Курбан Администратор',
            '2026-04-01 09:00:00+05'::timestamptz
        ),
        (
            '86f9d985-31d2-40e2-b466-706566bff629'::uuid,
            'demo-google-sub-solovyov',
            'a.solovyov@goldencompass.kz',
            'Алексей Соловьев',
            '2026-04-01 09:05:00+05'::timestamptz
        ),
        (
            'e912e9d0-a2ab-48cf-bd0a-b04cd54ca8ff'::uuid,
            'demo-google-sub-madina',
            'madina@goldencompass.kz',
            'Мадина Ахметова',
            '2026-04-02 09:10:00+05'::timestamptz
        ),
        (
            'f5c7f36b-39e8-4ee4-bc37-b54a29f5426a'::uuid,
            'demo-google-sub-timur',
            'timur@goldencompass.kz',
            'Тимур Ибраев',
            '2026-04-02 09:15:00+05'::timestamptz
        )
)
insert into users (id, google_sub, email, full_name, created_at)
select id, google_sub, email, full_name, created_at
from user_seed
on conflict (id) do update set
    google_sub = excluded.google_sub,
    email = excluded.email,
    full_name = excluded.full_name;

insert into admin_access (email)
values
    ('kurban@goldencompass.kz'),
    ('kitchen@goldencompass.kz'),
    ('a.solovyov@goldencompass.kz')
on conflict (email) do update
set revoked_at = null;

insert into report_runs (period_start, period_end, sent_at)
values ('2026-04-01'::date, '2026-04-30'::date, '2026-04-30 18:00:00+05'::timestamptz)
on conflict (period_start, period_end) do update
set sent_at = excluded.sent_at;

with user_seed (user_id, employee_index, device_id, external_ip) as (
    values
        ('561a0ae5-679e-4643-a7c4-bf451ace46b4'::uuid, 0, 'demo-device-kitchen-01', '91.185.9.10'),
        ('0c450c7f-9cf0-4323-87f0-d3bfbd332a41'::uuid, 1, 'demo-device-kurban-01', '91.185.9.11'),
        ('86f9d985-31d2-40e2-b466-706566bff629'::uuid, 2, 'demo-device-solovyov-01', '91.185.9.12'),
        ('e912e9d0-a2ab-48cf-bd0a-b04cd54ca8ff'::uuid, 3, 'demo-device-madina-01', '91.185.9.14'),
        ('f5c7f36b-39e8-4ee4-bc37-b54a29f5426a'::uuid, 4, 'demo-device-timur-01', '91.185.9.13')
),
calendar_days as (
    select generate_series('2026-04-01'::date, '2026-05-31'::date, interval '1 day')::date as business_date
),
seed as (
    select
        us.user_id,
        us.employee_index,
        us.device_id,
        us.external_ip,
        cd.business_date,
        extract(isodow from cd.business_date)::int as iso_day,
        extract(day from cd.business_date)::int as month_day
    from user_seed us
    cross join calendar_days cd
),
attendance_seed as (
    select
        user_id,
        business_date,
        device_id,
        external_ip,
        (
            case
                when employee_index = 0 and month_day in (4, 12, 19, 28) then time '08:25'
                when employee_index = 1 and month_day in (3, 11, 20) then time '08:05'
                when employee_index = 2 and month_day in (8, 15, 22) then time '09:10'
                when employee_index = 3 and month_day in (2, 9, 16, 23) then time '08:45'
                when employee_index = 4 and month_day in (6, 17, 29) then time '10:00'
                else time '08:00'
            end
        ) as check_in_time,
        (
            case
                when employee_index = 0 and month_day in (7, 15, 26) then time '16:35'
                when employee_index = 1 and month_day in (13, 27) then time '16:45'
                when employee_index = 2 and month_day in (5, 19, 30) then time '18:20'
                when employee_index = 3 and month_day in (10, 21) then time '15:55'
                when employee_index = 4 and month_day in (12, 24) then null
                else time '17:00'
            end
        ) as check_out_time
    from seed
    where iso_day <= 5
        and not (
            employee_index = 4
            and month_day in (3, 14, 25)
        )
)
insert into attendance_records (user_id, business_date)
select user_id, business_date
from attendance_seed
on conflict (user_id, business_date) do nothing;

with user_seed (user_id, employee_index, device_id, external_ip) as (
    values
        ('561a0ae5-679e-4643-a7c4-bf451ace46b4'::uuid, 0, 'demo-device-kitchen-01', '91.185.9.10'),
        ('0c450c7f-9cf0-4323-87f0-d3bfbd332a41'::uuid, 1, 'demo-device-kurban-01', '91.185.9.11'),
        ('86f9d985-31d2-40e2-b466-706566bff629'::uuid, 2, 'demo-device-solovyov-01', '91.185.9.12'),
        ('e912e9d0-a2ab-48cf-bd0a-b04cd54ca8ff'::uuid, 3, 'demo-device-madina-01', '91.185.9.14'),
        ('f5c7f36b-39e8-4ee4-bc37-b54a29f5426a'::uuid, 4, 'demo-device-timur-01', '91.185.9.13')
),
calendar_days as (
    select generate_series('2026-04-01'::date, '2026-05-31'::date, interval '1 day')::date as business_date
),
seed as (
    select
        us.user_id,
        us.employee_index,
        us.device_id,
        us.external_ip,
        cd.business_date,
        extract(isodow from cd.business_date)::int as iso_day,
        extract(day from cd.business_date)::int as month_day
    from user_seed us
    cross join calendar_days cd
),
attendance_seed as (
    select
        user_id,
        business_date,
        case
            when business_date = '2026-05-12' and employee_index = 0 then '2026-05-12 08:10:00+05'::timestamptz
            when business_date = '2026-05-12' and employee_index = 3 then '2026-05-12 08:25:00+05'::timestamptz
            else (
                business_date +
                case
                    when employee_index = 0 and month_day in (4, 12, 19, 28) then time '08:25'
                    when employee_index = 1 and month_day in (3, 11, 20) then time '08:05'
                    when employee_index = 2 and month_day in (8, 15, 22) then time '09:10'
                    when employee_index = 3 and month_day in (2, 9, 16, 23) then time '08:45'
                    when employee_index = 4 and month_day in (6, 17, 29) then time '10:00'
                    else time '08:00'
                end
            ) at time zone 'Asia/Almaty'
        end as check_in_at,
        (
            case
                when employee_index = 0 and month_day in (7, 15, 26) then time '16:35'
                when employee_index = 1 and month_day in (13, 27) then time '16:45'
                when employee_index = 2 and month_day in (5, 19, 30) then time '18:20'
                when employee_index = 3 and month_day in (10, 21) then time '15:55'
                when employee_index = 4 and month_day in (12, 24) then null
                else time '17:00'
            end
        ) as check_out_time,
        case
            -- kitchen marks from Madina's device
            when business_date = '2026-05-20' and employee_index = 0 then 'demo-device-madina-01'
            -- Madina marks from kitchen's device
            when business_date = '2026-05-23' and employee_index = 3 then 'demo-device-kitchen-01'
            -- Timur marks from Solovyov's device
            when business_date = '2026-05-29' and employee_index = 4 then 'demo-device-solovyov-01'
            -- April archive also has one device case
            when business_date = '2026-04-17' and employee_index = 2 then 'demo-device-kurban-01'
            else device_id
        end as device_id,
        case
            -- exact same office Wi-Fi/NAT within a short interval
            when business_date = '2026-05-12' and employee_index in (0, 3) then '91.185.9.200'
            -- same mobile hotspot / shared network during check-in
            when business_date = '2026-05-22' and employee_index in (2, 4) then '91.185.9.201'
            -- archive month example
            when business_date = '2026-04-18' and employee_index in (1, 3) then '91.185.9.202'
            else external_ip
        end as external_ip
    from seed
    where iso_day <= 5
        and not (
            employee_index = 4
            and month_day in (3, 14, 25)
        )
)
insert into attendance_events (
    record_id,
    event_type,
    event_at,
    status,
    phone_model,
    browser,
    device_id,
    external_ip
)
select
    ar.id,
    'check_in',
    seed.check_in_at,
    'normal',
    'Demo Phone',
    'Chrome',
    seed.device_id,
    seed.external_ip
from attendance_seed seed
join attendance_records ar
    on ar.user_id = seed.user_id
    and ar.business_date = seed.business_date
on conflict (record_id, event_type) do update set
    event_at = excluded.event_at,
    status = excluded.status,
    phone_model = excluded.phone_model,
    browser = excluded.browser,
    device_id = excluded.device_id,
    external_ip = excluded.external_ip;

with user_seed (user_id, employee_index, device_id, external_ip) as (
    values
        ('561a0ae5-679e-4643-a7c4-bf451ace46b4'::uuid, 0, 'demo-device-kitchen-01', '91.185.9.10'),
        ('0c450c7f-9cf0-4323-87f0-d3bfbd332a41'::uuid, 1, 'demo-device-kurban-01', '91.185.9.11'),
        ('86f9d985-31d2-40e2-b466-706566bff629'::uuid, 2, 'demo-device-solovyov-01', '91.185.9.12'),
        ('e912e9d0-a2ab-48cf-bd0a-b04cd54ca8ff'::uuid, 3, 'demo-device-madina-01', '91.185.9.14'),
        ('f5c7f36b-39e8-4ee4-bc37-b54a29f5426a'::uuid, 4, 'demo-device-timur-01', '91.185.9.13')
),
calendar_days as (
    select generate_series('2026-04-01'::date, '2026-05-31'::date, interval '1 day')::date as business_date
),
seed as (
    select
        us.user_id,
        us.employee_index,
        us.device_id,
        us.external_ip,
        cd.business_date,
        extract(isodow from cd.business_date)::int as iso_day,
        extract(day from cd.business_date)::int as month_day
    from user_seed us
    cross join calendar_days cd
),
attendance_seed as (
    select
        user_id,
        business_date,
        (
            business_date +
            case
                when employee_index = 0 and month_day in (7, 15, 26) then time '16:35'
                when employee_index = 1 and month_day in (13, 27) then time '16:45'
                when employee_index = 2 and month_day in (5, 19, 30) then time '18:20'
                when employee_index = 3 and month_day in (10, 21) then time '15:55'
                when employee_index = 4 and month_day in (12, 24) then null
                else time '17:00'
            end
        ) at time zone 'Asia/Almaty' as check_out_at,
        case
            -- kitchen check-out from Kurban's device
            when business_date = '2026-05-26' and employee_index = 0 then 'demo-device-kurban-01'
            -- Madina check-out from Timur's device
            when business_date = '2026-05-21' and employee_index = 3 then 'demo-device-timur-01'
            else device_id
        end as device_id,
        case
            -- same network during leaving time
            when business_date = '2026-05-15' and employee_index in (0, 1) then '91.185.9.203'
            -- archive month leaving-time overlap
            when business_date = '2026-04-24' and employee_index in (2, 3) then '91.185.9.204'
            else external_ip
        end as external_ip
    from seed
    where iso_day <= 5
        and not (
            employee_index = 4
            and month_day in (3, 14, 25)
        )
)
insert into attendance_events (
    record_id,
    event_type,
    event_at,
    status,
    phone_model,
    browser,
    device_id,
    external_ip
)
select
    ar.id,
    'check_out',
    seed.check_out_at,
    'normal',
    'Demo Phone',
    'Chrome',
    seed.device_id,
    seed.external_ip
from attendance_seed seed
join attendance_records ar
    on ar.user_id = seed.user_id
    and ar.business_date = seed.business_date
where seed.check_out_at is not null
on conflict (record_id, event_type) do update set
    event_at = excluded.event_at,
    status = excluded.status,
    phone_model = excluded.phone_model,
    browser = excluded.browser,
    device_id = excluded.device_id,
    external_ip = excluded.external_ip;
