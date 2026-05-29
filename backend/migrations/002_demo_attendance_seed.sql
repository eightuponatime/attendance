-- Demo seed for manual execution in DBeaver.
-- It creates a visually varied May 2026 attendance month for:
-- kitchen@goldencompass.kz / 561a0ae5-679e-4643-a7c4-bf451ace46b4
--
-- Safe to run multiple times: records/events are upserted.

insert into users (id, google_sub, email, full_name, created_at)
values (
    '561a0ae5-679e-4643-a7c4-bf451ace46b4',
    '103609409226151733153',
    'kitchen@goldencompass.kz',
    'Столовая Отчеты по столовой',
    '2026-05-16 17:41:05.722+05'
)
on conflict (id) do update set
    google_sub = excluded.google_sub,
    email = excluded.email,
    full_name = excluded.full_name;

with seed (
    business_date,
    check_in_at,
    check_out_at
) as (
    values
        -- on time
        ('2026-05-01'::date, '2026-05-01 08:00:00+05'::timestamptz, '2026-05-01 17:00:00+05'::timestamptz),
        -- weekend short day
        ('2026-05-02'::date, '2026-05-02 10:00:00+05'::timestamptz, '2026-05-02 15:00:00+05'::timestamptz),
        -- weekend full day
        ('2026-05-03'::date, '2026-05-03 09:00:00+05'::timestamptz, '2026-05-03 18:00:00+05'::timestamptz),
        -- late arrival and early leave
        ('2026-05-04'::date, '2026-05-04 08:20:00+05'::timestamptz, '2026-05-04 16:30:00+05'::timestamptz),
        -- early arrival and late leave
        ('2026-05-05'::date, '2026-05-05 07:45:00+05'::timestamptz, '2026-05-05 17:20:00+05'::timestamptz),
        -- slightly late, slightly under target
        ('2026-05-06'::date, '2026-05-06 08:05:00+05'::timestamptz, '2026-05-06 17:02:00+05'::timestamptz),
        -- overtime
        ('2026-05-07'::date, '2026-05-07 08:00:00+05'::timestamptz, '2026-05-07 18:10:00+05'::timestamptz),
        -- big late arrival
        ('2026-05-08'::date, '2026-05-08 09:15:00+05'::timestamptz, '2026-05-08 17:00:00+05'::timestamptz),
        -- weekend half day
        ('2026-05-09'::date, '2026-05-09 08:30:00+05'::timestamptz, '2026-05-09 12:00:00+05'::timestamptz),
        -- early arrival, exact target duration
        ('2026-05-10'::date, '2026-05-10 07:50:00+05'::timestamptz, '2026-05-10 16:50:00+05'::timestamptz),
        -- current stats week: good baseline
        ('2026-05-11'::date, '2026-05-11 08:00:00+05'::timestamptz, '2026-05-11 17:00:00+05'::timestamptz),
        -- late and under target
        ('2026-05-12'::date, '2026-05-12 08:42:00+05'::timestamptz, '2026-05-12 17:20:00+05'::timestamptz),
        -- early and overtime
        ('2026-05-13'::date, '2026-05-13 07:35:00+05'::timestamptz, '2026-05-13 17:05:00+05'::timestamptz),
        -- late but compensated with late leave
        ('2026-05-14'::date, '2026-05-14 09:05:00+05'::timestamptz, '2026-05-14 19:00:00+05'::timestamptz),
        -- late and early leave
        ('2026-05-15'::date, '2026-05-15 08:10:00+05'::timestamptz, '2026-05-15 15:55:00+05'::timestamptz),
        -- in progress: check-in exists, check-out does not
        ('2026-05-16'::date, '2026-05-16 08:25:00+05'::timestamptz, null::timestamptz),
        -- 2026-05-17 intentionally omitted to show an empty gray day
        ('2026-05-18'::date, '2026-05-18 08:00:00+05'::timestamptz, '2026-05-18 17:30:00+05'::timestamptz),
        ('2026-05-19'::date, '2026-05-19 10:00:00+05'::timestamptz, '2026-05-19 17:00:00+05'::timestamptz),
        ('2026-05-20'::date, '2026-05-20 07:55:00+05'::timestamptz, '2026-05-20 16:30:00+05'::timestamptz),
        ('2026-05-21'::date, '2026-05-21 08:00:00+05'::timestamptz, '2026-05-21 18:00:00+05'::timestamptz),
        ('2026-05-22'::date, '2026-05-22 08:50:00+05'::timestamptz, '2026-05-22 18:10:00+05'::timestamptz),
        ('2026-05-23'::date, '2026-05-23 11:00:00+05'::timestamptz, '2026-05-23 14:00:00+05'::timestamptz),
        ('2026-05-24'::date, '2026-05-24 07:40:00+05'::timestamptz, '2026-05-24 17:00:00+05'::timestamptz),
        ('2026-05-25'::date, '2026-05-25 08:15:00+05'::timestamptz, '2026-05-25 17:00:00+05'::timestamptz),
        ('2026-05-26'::date, '2026-05-26 08:00:00+05'::timestamptz, '2026-05-26 16:45:00+05'::timestamptz),
        ('2026-05-27'::date, '2026-05-27 07:50:00+05'::timestamptz, '2026-05-27 18:25:00+05'::timestamptz),
        ('2026-05-28'::date, '2026-05-28 09:30:00+05'::timestamptz, '2026-05-28 17:30:00+05'::timestamptz),
        ('2026-05-29'::date, '2026-05-29 08:00:00+05'::timestamptz, '2026-05-29 17:00:00+05'::timestamptz),
        ('2026-05-30'::date, '2026-05-30 12:00:00+05'::timestamptz, '2026-05-30 16:00:00+05'::timestamptz),
        ('2026-05-31'::date, '2026-05-31 08:10:00+05'::timestamptz, '2026-05-31 17:40:00+05'::timestamptz)
)
insert into attendance_records (user_id, business_date)
select
    '561a0ae5-679e-4643-a7c4-bf451ace46b4',
    seed.business_date
from seed
on conflict (user_id, business_date) do nothing;

with seed (
    business_date,
    check_in_at,
    check_out_at
) as (
    values
        ('2026-05-01'::date, '2026-05-01 08:00:00+05'::timestamptz, '2026-05-01 17:00:00+05'::timestamptz),
        ('2026-05-02'::date, '2026-05-02 10:00:00+05'::timestamptz, '2026-05-02 15:00:00+05'::timestamptz),
        ('2026-05-03'::date, '2026-05-03 09:00:00+05'::timestamptz, '2026-05-03 18:00:00+05'::timestamptz),
        ('2026-05-04'::date, '2026-05-04 08:20:00+05'::timestamptz, '2026-05-04 16:30:00+05'::timestamptz),
        ('2026-05-05'::date, '2026-05-05 07:45:00+05'::timestamptz, '2026-05-05 17:20:00+05'::timestamptz),
        ('2026-05-06'::date, '2026-05-06 08:05:00+05'::timestamptz, '2026-05-06 17:02:00+05'::timestamptz),
        ('2026-05-07'::date, '2026-05-07 08:00:00+05'::timestamptz, '2026-05-07 18:10:00+05'::timestamptz),
        ('2026-05-08'::date, '2026-05-08 09:15:00+05'::timestamptz, '2026-05-08 17:00:00+05'::timestamptz),
        ('2026-05-09'::date, '2026-05-09 08:30:00+05'::timestamptz, '2026-05-09 12:00:00+05'::timestamptz),
        ('2026-05-10'::date, '2026-05-10 07:50:00+05'::timestamptz, '2026-05-10 16:50:00+05'::timestamptz),
        ('2026-05-11'::date, '2026-05-11 08:00:00+05'::timestamptz, '2026-05-11 17:00:00+05'::timestamptz),
        ('2026-05-12'::date, '2026-05-12 08:42:00+05'::timestamptz, '2026-05-12 17:20:00+05'::timestamptz),
        ('2026-05-13'::date, '2026-05-13 07:35:00+05'::timestamptz, '2026-05-13 17:05:00+05'::timestamptz),
        ('2026-05-14'::date, '2026-05-14 09:05:00+05'::timestamptz, '2026-05-14 19:00:00+05'::timestamptz),
        ('2026-05-15'::date, '2026-05-15 08:10:00+05'::timestamptz, '2026-05-15 15:55:00+05'::timestamptz),
        ('2026-05-16'::date, '2026-05-16 08:25:00+05'::timestamptz, null::timestamptz),
        ('2026-05-18'::date, '2026-05-18 08:00:00+05'::timestamptz, '2026-05-18 17:30:00+05'::timestamptz),
        ('2026-05-19'::date, '2026-05-19 10:00:00+05'::timestamptz, '2026-05-19 17:00:00+05'::timestamptz),
        ('2026-05-20'::date, '2026-05-20 07:55:00+05'::timestamptz, '2026-05-20 16:30:00+05'::timestamptz),
        ('2026-05-21'::date, '2026-05-21 08:00:00+05'::timestamptz, '2026-05-21 18:00:00+05'::timestamptz),
        ('2026-05-22'::date, '2026-05-22 08:50:00+05'::timestamptz, '2026-05-22 18:10:00+05'::timestamptz),
        ('2026-05-23'::date, '2026-05-23 11:00:00+05'::timestamptz, '2026-05-23 14:00:00+05'::timestamptz),
        ('2026-05-24'::date, '2026-05-24 07:40:00+05'::timestamptz, '2026-05-24 17:00:00+05'::timestamptz),
        ('2026-05-25'::date, '2026-05-25 08:15:00+05'::timestamptz, '2026-05-25 17:00:00+05'::timestamptz),
        ('2026-05-26'::date, '2026-05-26 08:00:00+05'::timestamptz, '2026-05-26 16:45:00+05'::timestamptz),
        ('2026-05-27'::date, '2026-05-27 07:50:00+05'::timestamptz, '2026-05-27 18:25:00+05'::timestamptz),
        ('2026-05-28'::date, '2026-05-28 09:30:00+05'::timestamptz, '2026-05-28 17:30:00+05'::timestamptz),
        ('2026-05-29'::date, '2026-05-29 08:00:00+05'::timestamptz, '2026-05-29 17:00:00+05'::timestamptz),
        ('2026-05-30'::date, '2026-05-30 12:00:00+05'::timestamptz, '2026-05-30 16:00:00+05'::timestamptz),
        ('2026-05-31'::date, '2026-05-31 08:10:00+05'::timestamptz, '2026-05-31 17:40:00+05'::timestamptz)
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
    'demo-device-kitchen-01',
    '127.0.0.1'
from seed
join attendance_records ar
    on ar.user_id = '561a0ae5-679e-4643-a7c4-bf451ace46b4'
    and ar.business_date = seed.business_date
on conflict (record_id, event_type) do update set
    event_at = excluded.event_at,
    status = excluded.status,
    phone_model = excluded.phone_model,
    browser = excluded.browser,
    device_id = excluded.device_id,
    external_ip = excluded.external_ip;

with seed (
    business_date,
    check_out_at
) as (
    values
        ('2026-05-01'::date, '2026-05-01 17:00:00+05'::timestamptz),
        ('2026-05-02'::date, '2026-05-02 15:00:00+05'::timestamptz),
        ('2026-05-03'::date, '2026-05-03 18:00:00+05'::timestamptz),
        ('2026-05-04'::date, '2026-05-04 16:30:00+05'::timestamptz),
        ('2026-05-05'::date, '2026-05-05 17:20:00+05'::timestamptz),
        ('2026-05-06'::date, '2026-05-06 17:02:00+05'::timestamptz),
        ('2026-05-07'::date, '2026-05-07 18:10:00+05'::timestamptz),
        ('2026-05-08'::date, '2026-05-08 17:00:00+05'::timestamptz),
        ('2026-05-09'::date, '2026-05-09 12:00:00+05'::timestamptz),
        ('2026-05-10'::date, '2026-05-10 16:50:00+05'::timestamptz),
        ('2026-05-11'::date, '2026-05-11 17:00:00+05'::timestamptz),
        ('2026-05-12'::date, '2026-05-12 17:20:00+05'::timestamptz),
        ('2026-05-13'::date, '2026-05-13 17:05:00+05'::timestamptz),
        ('2026-05-14'::date, '2026-05-14 19:00:00+05'::timestamptz),
        ('2026-05-15'::date, '2026-05-15 15:55:00+05'::timestamptz),
        ('2026-05-18'::date, '2026-05-18 17:30:00+05'::timestamptz),
        ('2026-05-19'::date, '2026-05-19 17:00:00+05'::timestamptz),
        ('2026-05-20'::date, '2026-05-20 16:30:00+05'::timestamptz),
        ('2026-05-21'::date, '2026-05-21 18:00:00+05'::timestamptz),
        ('2026-05-22'::date, '2026-05-22 18:10:00+05'::timestamptz),
        ('2026-05-23'::date, '2026-05-23 14:00:00+05'::timestamptz),
        ('2026-05-24'::date, '2026-05-24 17:00:00+05'::timestamptz),
        ('2026-05-25'::date, '2026-05-25 17:00:00+05'::timestamptz),
        ('2026-05-26'::date, '2026-05-26 16:45:00+05'::timestamptz),
        ('2026-05-27'::date, '2026-05-27 18:25:00+05'::timestamptz),
        ('2026-05-28'::date, '2026-05-28 17:30:00+05'::timestamptz),
        ('2026-05-29'::date, '2026-05-29 17:00:00+05'::timestamptz),
        ('2026-05-30'::date, '2026-05-30 16:00:00+05'::timestamptz),
        ('2026-05-31'::date, '2026-05-31 17:40:00+05'::timestamptz)
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
    'demo-device-kitchen-01',
    '127.0.0.1'
from seed
join attendance_records ar
    on ar.user_id = '561a0ae5-679e-4643-a7c4-bf451ace46b4'
    and ar.business_date = seed.business_date
on conflict (record_id, event_type) do update set
    event_at = excluded.event_at,
    status = excluded.status,
    phone_model = excluded.phone_model,
    browser = excluded.browser,
    device_id = excluded.device_id,
    external_ip = excluded.external_ip;
