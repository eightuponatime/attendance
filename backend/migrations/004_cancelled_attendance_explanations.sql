do $$
begin
    if exists (
        select 1
        from pg_constraint
        where conname = 'attendance_explanations_status_check'
    ) then
        alter table attendance_explanations
            drop constraint attendance_explanations_status_check;
    end if;

    alter table attendance_explanations
        add constraint attendance_explanations_status_check
        check (status in ('pending', 'approved', 'rejected', 'cancelled'));
end $$;
