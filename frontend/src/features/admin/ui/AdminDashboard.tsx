import { AlertTriangle, ArrowLeft, CalendarDays, ChevronDown, LogOut, Radio, Search, ShieldCheck } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  addAdminAccess,
  getAdminEmployee,
  getAdminEmployees,
  getAdminAccess,
  getAdminReports,
  getAdminSessions,
  getAdminSuspiciousActivity,
  getAdminSystemOutages,
  getAdminOutageDay,
  isAdminAuthError,
  repairAdminOutage,
  revokeAdminAccess,
  revokeAdminSession,
} from "../api/adminApi";
import { errorText } from "../../../shared/api/errors";
import { ConfirmDialog } from "../../../shared/ui/ConfirmDialog";
import type {
  AdminAccess,
  AdminEmployeeMonthDetail,
  AdminEmployeesMonth,
  AdminEmployeeSummary,
  AdminReportRun,
  AdminSession,
  AdminSuspiciousActivity,
  AdminSuspiciousDeviceMatch,
  AdminSuspiciousIPMatch,
  AdminSystemOutage,
  AdminOutageDay,
  AdminOutageRepairItem,
  AttendanceDaySummary,
  AdminMe,
} from "../../../shared/types/api";

type EmployeeSuspiciousActivity = {
  deviceMatches: AdminSuspiciousDeviceMatch[];
  ipMatches: AdminSuspiciousIPMatch[];
  total: number;
};

type AdminPageTab = "employees" | "access" | "outages";
type EmployeeFilter = "all" | "late" | "early";

export function AdminDashboard({
  user,
  onLogout,
  onAuthLost,
}: {
  user: AdminMe | null;
  onLogout: () => void;
  onAuthLost: () => void;
}) {
  const [month, setMonth] = useState(() => monthFromURL() ?? currentMonth());
  const [employees, setEmployees] = useState<AdminEmployeesMonth | null>(null);
  const [reports, setReports] = useState<AdminReportRun[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedEmployee, setSelectedEmployee] = useState<AdminEmployeeMonthDetail | null>(null);
  const [suspicious, setSuspicious] = useState<AdminSuspiciousActivity | null>(null);
  const [outages, setOutages] = useState<AdminSystemOutage[]>([]);
  const [pageTab, setPageTab] = useState<AdminPageTab>("employees");
  const [query, setQuery] = useState("");
  const [employeeFilter, setEmployeeFilter] = useState<EmployeeFilter>("all");
  const [mobileDetailOpen, setMobileDetailOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const isLive = month === currentMonth();
  const handleError = useCallback((err: unknown) => {
    if (isAdminAuthError(err)) {
      onAuthLost();
      return;
    }
    setError(errorText(err));
  }, [onAuthLost]);

  useEffect(() => {
    getAdminReports()
      .then((data) => setReports(data.items))
      .catch((err: unknown) => {
        if (isAdminAuthError(err)) {
          onAuthLost();
          return;
        }
        setReports([]);
      });
  }, [onAuthLost]);

  useEffect(() => {
    let alive = true;

    const load = () => {
      Promise.all([getAdminEmployees(month), getAdminSuspiciousActivity(month), getAdminSystemOutages(month)])
        .then(([employeeData, suspiciousData, outageData]) => {
          if (!alive) return;
          setEmployees(employeeData);
          setSuspicious(suspiciousData);
          setOutages(outageData.items);
          setSelectedUserId((current) => current ?? employeeData.employees[0]?.user_id ?? null);
          setError(null);
        })
        .catch((err: unknown) => {
          if (alive) handleError(err);
        });
    };

    load();
    if (!isLive) {
      return () => {
        alive = false;
      };
    }

    const interval = window.setInterval(load, 60_000);
    return () => {
      alive = false;
      window.clearInterval(interval);
    };
  }, [handleError, isLive, month]);

  useEffect(() => {
    if (!selectedUserId) {
      setSelectedEmployee(null);
      return;
    }

    getAdminEmployee(selectedUserId, month)
      .then((data) => {
        setSelectedEmployee(data);
        setError(null);
      })
      .catch(handleError);
  }, [handleError, month, selectedUserId]);

  const suspiciousCounts = useMemo(() => suspiciousCountsByUser(suspicious), [suspicious]);
  const filteredEmployees = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    const rows = employees?.employees ?? [];

    return rows.filter((employee) => {
      const matchesQuery =
        normalizedQuery === "" ||
        `${employee.full_name} ${employee.email}`.toLowerCase().includes(normalizedQuery);
      if (!matchesQuery) return false;

      switch (employeeFilter) {
        case "late":
          return employee.late_count > 0;
        case "early":
          return employee.early_leave_count > 0;
        case "all":
          return true;
      }
    });
  }, [employeeFilter, employees, query, suspiciousCounts]);

  return (
    <section className="admin-page">
      <header className="admin-topbar">
        <div>
          <span>Панель HR</span>
          <h1>Учет рабочего времени</h1>
          {user && <p>{user.full_name}</p>}
        </div>

        <div className="admin-topbar-actions">
          <div className="admin-period-controls">
            {isLive && (
              <button className="live-button live-button-active" type="button">
                <Radio size={16} />
                LIVE
              </button>
            )}

            <MonthPicker reports={reports} value={month} onChange={setMonth} />
          </div>

          <button
            className="admin-logout-button"
            type="button"
            aria-label="Выйти из админ-панели"
            title="Выйти"
            onClick={onLogout}
          >
            <LogOut size={18} />
          </button>
        </div>
      </header>

      {error && <p className="error-banner">{error}</p>}

      <div className="admin-page-tabs">
        <button
          className={pageTab === "employees" ? "admin-page-tab-active" : ""}
          type="button"
          onClick={() => setPageTab("employees")}
        >
          Сотрудники
        </button>
        <button
          className={pageTab === "outages" ? "admin-page-tab-active" : ""}
          type="button"
          onClick={() => setPageTab("outages")}
        >
          Сбои сервера
        </button>
        <button
          className={pageTab === "access" ? "admin-page-tab-active" : ""}
          type="button"
          onClick={() => setPageTab("access")}
        >
          Управление доступом
        </button>
      </div>

      {pageTab === "employees" ? (
        <div className={`admin-layout ${mobileDetailOpen ? "admin-layout-detail-open" : ""}`}>
          <section className="admin-panel employee-panel">
            <div className="admin-panel-header">
              <div>
                <span>{isLive ? "Текущий месяц" : "Архивный отчет"}</span>
                <h2>{monthTitle(month)}</h2>
              </div>
              <ShieldCheck size={22} />
            </div>

            <label className="admin-search">
              <Search size={16} />
              <input
                value={query}
                placeholder="Поиск сотрудника"
                onChange={(event) => setQuery(event.target.value)}
              />
            </label>

            <EmployeeFilters
              active={employeeFilter}
              employees={employees?.employees ?? []}
              onChange={setEmployeeFilter}
            />

            <p className="admin-list-count">
              Показано {filteredEmployees.length} из {employees?.employees.length ?? 0}
            </p>

            <div className="employee-list">
              {filteredEmployees.map((employee) => (
                <EmployeeRow
                  key={employee.user_id}
                  employee={employee}
                  suspiciousCount={suspiciousCounts.get(employee.user_id) ?? 0}
                  active={employee.user_id === selectedUserId}
                  onClick={() => {
                    setSelectedUserId(employee.user_id);
                    setMobileDetailOpen(true);
                  }}
                />
              ))}
            </div>
          </section>

          <section className="admin-detail-stack">
            <button
              className="admin-detail-back"
              type="button"
              onClick={() => setMobileDetailOpen(false)}
            >
              <ArrowLeft size={18} />
              К списку сотрудников
            </button>
            {selectedEmployee ? (
              <EmployeeDetail employee={selectedEmployee} suspicious={suspicious} />
            ) : (
              <section className="admin-panel">
                <p className="muted-text">Выберите сотрудника</p>
              </section>
            )}
          </section>
        </div>
      ) : pageTab === "outages" ? (
        <OutagesPanel outages={outages} onAuthLost={onAuthLost} />
      ) : (
        <AccessManagement onAuthLost={onAuthLost} />
      )}
    </section>
  );
}

function MonthPicker({
  reports,
  value,
  onChange,
}: {
  reports: AdminReportRun[];
  value: string;
  onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const options = useMemo(() => monthOptions(reports), [reports]);
  const years = useMemo(() => uniqueYears(options), [options]);
  const [year, setYear] = useState(() => value.slice(0, 4));
  const visibleOptions = options.filter((option) => option.value.startsWith(`${year}-`));

  useEffect(() => {
    setYear(value.slice(0, 4));
  }, [value]);

  return (
    <div className="month-picker">
      <button
        className="month-picker-button"
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
      >
        <CalendarDays size={16} />
        <span>{value === currentMonth() ? "Текущий месяц" : monthTitle(value)}</span>
        <ChevronDown size={16} />
      </button>

      {open && (
        <div className="month-picker-layer" role="presentation" onMouseDown={() => setOpen(false)}>
          <div
            className="month-picker-popover"
            role="dialog"
            aria-modal="true"
            aria-label="Выбор месяца"
            onMouseDown={(event) => event.stopPropagation()}
          >
            <div className="month-picker-years">
              {years.map((item) => (
                <button
                  key={item}
                  className={item === year ? "month-picker-year-active" : ""}
                  type="button"
                  onClick={() => setYear(item)}
                >
                  {item}
                </button>
              ))}
            </div>

            <div className="month-picker-list">
              {visibleOptions.map((option) => (
                <button
                  key={option.value}
                  className={option.value === value ? "month-picker-option-active" : ""}
                  type="button"
                  onClick={() => {
                    onChange(option.value);
                    setOpen(false);
                  }}
                >
                  <span>{option.label}</span>
                  {option.value === currentMonth() && <strong>LIVE</strong>}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function EmployeeFilters({
  active,
  employees,
  onChange,
}: {
  active: EmployeeFilter;
  employees: AdminEmployeeSummary[];
  onChange: (value: EmployeeFilter) => void;
}) {
  const counts = employeeFilterCounts(employees);
  const items: Array<{ value: EmployeeFilter; label: string; count: number }> = [
    { value: "all", label: "Все", count: employees.length },
    { value: "late", label: "Опоздания", count: counts.late },
    { value: "early", label: "Ранние уходы", count: counts.early },
  ];

  return (
    <div className="employee-filters">
      {items.map((item) => (
        <button
          key={item.value}
          className={item.value === active ? "employee-filter-active" : ""}
          type="button"
          onClick={() => onChange(item.value)}
        >
          {item.label}
          <span>{item.count}</span>
        </button>
      ))}
    </div>
  );
}

function EmployeeRow({
  employee,
  suspiciousCount,
  active,
  onClick,
}: {
  employee: AdminEmployeeSummary;
  suspiciousCount: number;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button className={`employee-row ${active ? "employee-row-active" : ""}`} type="button" onClick={onClick}>
      <div>
        <strong>{employee.full_name}</strong>
        <span>{employee.email}</span>
      </div>
      <div className="employee-chips">
        <Chip label="Часы" value={minutesToHours(employee.worked_minutes)} tone="neutral" />
        <Chip label="Подозр." value={suspiciousCount} tone={suspiciousCount > 0 ? "issue" : "ok"} />
        <Chip label="Опозд." value={employee.late_count} tone={employee.late_count > 0 ? "issue" : "ok"} />
        <Chip label="Ранний уход" value={employee.early_leave_count} tone={employee.early_leave_count > 0 ? "issue" : "ok"} />
        <Chip label="Нет ухода" value={employee.missing_check_outs} tone={employee.missing_check_outs > 0 ? "warn" : "ok"} />
      </div>
    </button>
  );
}

function Chip({
  label,
  value,
  tone,
}: {
  label: string;
  value: number | string;
  tone: "neutral" | "ok" | "warn" | "issue";
}) {
  return <span className={`admin-chip admin-chip-${tone}`}>{label}: {value}</span>;
}

function EmployeeDetail({
  employee,
  suspicious,
}: {
  employee: AdminEmployeeMonthDetail;
  suspicious: AdminSuspiciousActivity | null;
}) {
  const markedDays = employee.days.filter((day) => day.status !== "empty");
  const defaultDay = [...markedDays].reverse()[0] ?? employee.days.find((day) => day.status !== "empty") ?? employee.days[0];
  const [selectedDate, setSelectedDate] = useState(defaultDay?.date ?? "");
  const [tab, setTab] = useState<"calendar" | "suspicious">("calendar");

  useEffect(() => {
    setSelectedDate(defaultDay?.date ?? "");
  }, [defaultDay?.date, employee.user_id]);

  const selectedDay = employee.days.find((day) => day.date === selectedDate) ?? defaultDay;
  const suspiciousForEmployee = useMemo(
    () => filterSuspiciousForUser(suspicious, employee.user_id),
    [employee.user_id, suspicious],
  );

  return (
    <section className="admin-panel employee-detail">
      <div className="admin-panel-header">
        <div>
          <span>Сотрудник</span>
          <h2>{employee.full_name}</h2>
          <p>{employee.email}</p>
        </div>
        <strong>{minutesToHours(employee.worked_minutes)}</strong>
      </div>

      <div className="admin-metrics-row">
        <Metric label="Цель" value={minutesToHours(employee.target_minutes)} />
        <Metric label="Дней" value={String(employee.worked_days)} />
        <Metric label="Опозданий" value={String(employee.late_count)} issue={employee.late_count > 0} />
        <Metric label="Ранних уходов" value={String(employee.early_leave_count)} issue={employee.early_leave_count > 0} />
      </div>

      <div className="admin-tabs">
        <button
          className={tab === "calendar" ? "admin-tab-active" : ""}
          type="button"
          onClick={() => setTab("calendar")}
        >
          Календарь
        </button>
        <button
          className={tab === "suspicious" ? "admin-tab-active" : ""}
          type="button"
          onClick={() => setTab("suspicious")}
        >
          Подозрительная активность
          <span className={suspiciousForEmployee.total > 0 ? "admin-tab-count-issue" : "admin-tab-count-ok"}>
            {suspiciousForEmployee.total}
          </span>
        </button>
      </div>

      {tab === "calendar" ? (
        <>
          <AdminMonthCalendar
            days={employee.days}
            selectedDate={selectedDate}
            onSelectDay={setSelectedDate}
          />

          {selectedDay && <SelectedAdminDay day={selectedDay} />}
        </>
      ) : (
        <SuspiciousPanel activity={suspiciousForEmployee} />
      )}
    </section>
  );
}

function Metric({ label, value, issue }: { label: string; value: string; issue?: boolean }) {
  return (
    <div className="admin-metric">
      <span>{label}</span>
      <strong className={issue ? "metric-issue" : undefined}>{value}</strong>
    </div>
  );
}

function AdminMonthCalendar({
  days,
  selectedDate,
  onSelectDay,
}: {
  days: AttendanceDaySummary[];
  selectedDate: string;
  onSelectDay: (date: string) => void;
}) {
  const cells = calendarCells(days);
  return (
    <div className="admin-calendar">
      {["пн", "вт", "ср", "чт", "пт", "сб", "вс"].map((day) => (
        <span key={day} className="admin-calendar-weekday">
          {day}
        </span>
      ))}
      {cells.map((day, index) =>
        day ? (
          <button
            key={day.date}
            className={`admin-calendar-day ${day.date === selectedDate ? "admin-calendar-day-selected" : ""}`}
            type="button"
            onClick={() => onSelectDay(day.date)}
          >
            {Number(day.date.slice(8, 10))}
            {(day.status !== "empty" || day.impacted_by_outage) && <i className={dayDotClass(day)} />}
          </button>
        ) : (
          <span key={`blank-${index}`} />
        ),
      )}
    </div>
  );
}

function SelectedAdminDay({ day }: { day: AttendanceDaySummary }) {
  return (
    <div className="admin-day-card">
      <strong>{formatDayLong(day.date)}</strong>
      {day.impacted_by_outage && (
        <div className="admin-day-outage-note">
          <span>Сбой сервера</span>
          <b>HR может восстановить отметки после проверки</b>
        </div>
      )}
      <div>
        <span>Приход</span>
        <b>{day.check_in_at ? formatOnlyTime(day.check_in_at) : "Нет"}</b>
        {day.late_minutes > 0 && <em>Опоздание +{day.late_minutes} мин</em>}
      </div>
      <div>
        <span>Уход</span>
        <b>{day.check_out_at ? formatOnlyTime(day.check_out_at) : "Нет"}</b>
        {day.early_leave_minutes > 0 && <em>Ранний уход -{day.early_leave_minutes} мин</em>}
      </div>
      <div>
        <span>Отработано</span>
        <b>{minutesToClock(day.worked_minutes)}</b>
      </div>
    </div>
  );
}

function SuspiciousPanel({ activity }: { activity: EmployeeSuspiciousActivity }) {
  const deviceCount = activity.deviceMatches.length;
  const ipCount = activity.ipMatches.length;
  const rows = suspiciousRows(activity);

  return (
    <div className="suspicious-panel">
      {deviceCount === 0 && ipCount === 0 ? (
        <p className="muted-text">Совпадений по устройствам и IP не найдено</p>
      ) : (
        <>
          <div className="suspicious-card-list">
            {rows.map((row) => (
              <div key={row.id} className="suspicious-card">
                <div>
                  <strong>{row.date}</strong>
                  <span>{row.type}</span>
                </div>
                <p>{row.mark}</p>
                <em>{row.evidence}</em>
              </div>
            ))}
          </div>

          <div className="suspicious-table-wrap">
            <table className="suspicious-table">
              <thead>
                <tr>
                  <th>Дата</th>
                  <th>Тип</th>
                  <th>Отметка</th>
                  <th>Доказательство</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.id}>
                    <td>{row.date}</td>
                    <td>{row.type}</td>
                    <td>{row.mark}</td>
                    <td>{row.evidence}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}

function suspiciousRows(activity: EmployeeSuspiciousActivity): Array<{
  id: string;
  date: string;
  type: string;
  mark: string;
  evidence: string;
}> {
  const deviceRows = activity.deviceMatches.map((match) => ({
    id: `device-${match.event.event_id}`,
    date: formatDayShort(match.event.business_date),
    type: "Устройство",
    mark: `${formatOnlyTime(match.event.event_at)} · ${eventTypeText(match.event.event_type)}`,
    evidence: `Устройство ${shortId(match.device_id)} принадлежит: ${match.owner.full_name}`,
  }));

  const ipRows = activity.ipMatches.map((match) => ({
    id: `ip-${match.event.event_id}`,
    date: formatDayShort(match.event.business_date),
    type: "IP",
    mark: `${formatOnlyTime(match.event.event_at)} · ${eventTypeText(match.event.event_type)}`,
    evidence: `IP ${match.external_ip} за ${match.minutes_between} мин до этого был у: ${match.previous_event.full_name}`,
  }));

  return [...deviceRows, ...ipRows];
}

function OutagesPanel({
  outages,
  onAuthLost,
}: {
  outages: AdminSystemOutage[];
  onAuthLost: () => void;
}) {
  const impacted = outages.filter((outage) => outage.impacts_work_hours);
  const [selected, setSelected] = useState<AdminOutageDay | null>(null);
  const [draft, setDraft] = useState<Record<string, { check_in_at: string; check_out_at: string }>>({});
  const [resolutionNote, setResolutionNote] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [confirmRepairOpen, setConfirmRepairOpen] = useState(false);
  const [editingResolved, setEditingResolved] = useState(false);
  const [timePicker, setTimePicker] = useState<TimePickerTarget | null>(null);
  const repairDraft = useMemo(() => buildOutageRepairItems(draft, selected?.employees ?? []), [draft, selected]);
  const canSaveRepair = repairDraft.items.length > 0 && !repairDraft.hasInvalid && !repairDraft.hasCheckOutWithoutCheckIn && Boolean(resolutionNote.trim());
  const selectedResolved = Boolean(selected?.outage.resolved_at);
  const canEditSelected = !selectedResolved || editingResolved;

  const openOutage = async (outage: AdminSystemOutage) => {
    if (!outage.impacts_work_hours) return;
    try {
      const data = await getAdminOutageDay(outage.id);
      setSelected(data);
      setDraft({});
      setResolutionNote(data.outage.resolution_note ?? "");
      setConfirmRepairOpen(false);
      setEditingResolved(false);
      setTimePicker(null);
      setError(null);
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    }
  };

  const setDraftValue = (userId: string, field: "check_in_at" | "check_out_at", value: string) => {
    setDraft((current) => ({
      ...current,
      [userId]: {
        check_in_at: current[userId]?.check_in_at ?? "",
        check_out_at: current[userId]?.check_out_at ?? "",
        [field]: value,
      },
    }));
  };

  const fillStandardDay = () => {
    if (!selected) return;
    const next: Record<string, { check_in_at: string; check_out_at: string }> = {};
    for (const employee of selected.employees) {
      next[employee.user_id] = {
        check_in_at: employee.check_in_at ? "" : "8:00",
        check_out_at: employee.check_out_at ? "" : "17:00",
      };
    }
    setDraft(next);
    setResolutionNote((current) => current || "Заполнены пустые отметки стандартным рабочим днем 8:00-17:00 из-за сбоя сервера.");
  };

  const saveRepair = async () => {
    if (!selected) return;
    if (repairDraft.hasInvalid) {
      setError("Выберите время через селектор");
      return;
    }
    if (repairDraft.hasCheckOutWithoutCheckIn) {
      setError("Нельзя сохранить уход без прихода");
      return;
    }
    if (repairDraft.items.length === 0) {
      setError("Нет изменений для сохранения");
      return;
    }
    if (!resolutionNote.trim()) {
      setError("Добавьте комментарий к исправлению");
      return;
    }

    try {
      await repairAdminOutage(selected.outage.id, resolutionNote, repairDraft.items);
      const refreshed = await getAdminOutageDay(selected.outage.id);
      setSelected(refreshed);
      setDraft({});
      setConfirmRepairOpen(false);
      setEditingResolved(false);
      setError(null);
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    }
  };

  return (
    <div className="outage-layout">
      <section className="admin-panel outage-panel">
        <div className="admin-panel-header">
          <div>
            <span>Контроль доступности</span>
            <h2>Сбои сервера</h2>
            <p>Показываются только сбои, затронувшие выбранный месяц.</p>
          </div>
          <strong className="admin-count-badge">{impacted.length}</strong>
        </div>

        {error && <p className="error-banner">{error}</p>}

        {outages.length === 0 ? (
          <p className="muted-text">Сбоев за этот период не найдено</p>
        ) : (
          <div className="outage-list">
            {outages.map((outage) => (
              <button
                key={outage.id}
                className={`outage-row ${outage.impacts_work_hours ? "outage-row-impact" : ""} ${outage.resolved_at ? "outage-row-resolved" : ""}`}
                type="button"
                disabled={!outage.impacts_work_hours}
                onClick={() => void openOutage(outage)}
              >
                <div className="outage-icon">
                  <AlertTriangle size={18} />
                </div>
                <div>
                  <strong>{outage.affected_business_date ?? "Без влияния на бизнес-день"}</strong>
                  <span>
                    {formatDateTime(outage.started_at)} - {formatDateTime(outage.ended_at)}
                  </span>
                  <p>
                    {outage.resolved_at
                      ? `Исправлено${outage.resolved_by ? `: ${outage.resolved_by}` : ""}. Запись сохранена в архиве.`
                      : outage.impacts_work_hours
                      ? "Затронул рабочее окно 06:00-19:00. Данные за день могут быть неполными."
                      : "Не попал в рабочее окно."}
                  </p>
                </div>
                <em>{outage.resolved_at ? "Исправлено" : "Требует проверки"}</em>
              </button>
            ))}
          </div>
        )}
      </section>

      {selected && (
        <section className="admin-panel outage-repair-panel">
          <div className="admin-panel-header">
            <div>
              <span>Исправление дня</span>
              <h2>{selected.outage.affected_business_date}</h2>
              <p>Все изменения будут записаны в audit log с вашим email.</p>
            </div>
          </div>

          <p className={`outage-warning ${selectedResolved ? "outage-warning-resolved" : ""}`}>
            {selectedResolved
              ? "Этот сбой уже исправлен. Данные открыты в режиме архива; повторная правка требует отдельного действия."
              : "Вы изменяете посещаемость за день, затронутый сбоем сервера. Действие повлияет на статистику сотрудников."}
          </p>

          <div className="outage-actions">
            {!selectedResolved || editingResolved ? (
              <button type="button" onClick={fillStandardDay}>
                Заполнить пустые 8:00-17:00
              </button>
            ) : (
              <button type="button" onClick={() => setEditingResolved(true)}>
                Редактировать исправления
              </button>
            )}
            {editingResolved && (
              <button type="button" className="outage-secondary-button" onClick={() => { setEditingResolved(false); setDraft({}); }}>
                Отменить правку
              </button>
            )}
          </div>
          {repairDraft.hasCheckOutWithoutCheckIn && (
            <p className="outage-validation">Для ухода нужен приход: выберите приход или очистите уход.</p>
          )}

          <div className="outage-repair-list">
            {selected.employees.map((employee) => {
              const checkInValue = repairTimeValue(draft[employee.user_id]?.check_in_at, employee.check_in_at);
              const checkOutValue = repairTimeValue(draft[employee.user_id]?.check_out_at, employee.check_out_at);
              return (
                <div key={employee.user_id} className="outage-repair-row">
                  <div>
                    <strong>{employee.full_name}</strong>
                    <span>{employee.email}</span>
                  </div>
                  <label>
                    Приход
                    <TimePickerField
                      value={checkInValue}
                      disabled={!canEditSelected || (Boolean(employee.check_in_at) && !editingResolved)}
                      placeholder="8:00"
                      onOpen={() => setTimePicker({
                        userId: employee.user_id,
                        field: "check_in_at",
                        value: checkInValue,
                        defaultHour: "08",
                      })}
                    />
                  </label>
                  <label>
                    Уход
                    <TimePickerField
                      value={checkOutValue}
                      disabled={!canEditSelected || (Boolean(employee.check_out_at) && !editingResolved)}
                      placeholder="17:00"
                      onOpen={() => setTimePicker({
                        userId: employee.user_id,
                        field: "check_out_at",
                        value: checkOutValue,
                        defaultHour: "17",
                      })}
                    />
                  </label>
                </div>
              );
            })}
          </div>

          <label className="outage-note">
            Комментарий
            <textarea
              value={resolutionNote}
              rows={3}
              disabled={!canEditSelected}
              onChange={(event) => setResolutionNote(event.target.value)}
            />
          </label>

          {canEditSelected && (
            <button
              className="outage-save-button"
              type="button"
              disabled={!canSaveRepair}
              onClick={() => setConfirmRepairOpen(true)}
            >
              Сохранить изменения
            </button>
          )}
        </section>
      )}
      <TimePickerDialog
        target={timePicker}
        onCancel={() => setTimePicker(null)}
        onSelect={(value) => {
          if (!timePicker) return;
          setDraftValue(timePicker.userId, timePicker.field, value);
          setTimePicker(null);
        }}
      />
      <ConfirmDialog
        open={confirmRepairOpen}
        title="Сохранить исправления?"
        text="Отметки будут записаны в посещаемость, попадут в статистику сотрудников и сохранятся в audit log с вашим email."
        confirmText="Сохранить"
        tone="neutral"
        onConfirm={() => void saveRepair()}
        onCancel={() => setConfirmRepairOpen(false)}
      />
    </div>
  );
}

function AccessManagement({ onAuthLost }: { onAuthLost: () => void }) {
  const [access, setAccess] = useState<AdminAccess[]>([]);
  const [sessions, setSessions] = useState<AdminSession[]>([]);
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [confirmAction, setConfirmAction] = useState<
    | { kind: "access"; email: string }
    | { kind: "session"; session: AdminSession }
    | null
  >(null);
  const activeAccess = access.filter((item) => item.is_active);
  const activeSessions = sessions.filter((session) => session.is_active);
  const sessionByEmail = useMemo(() => latestSessionByEmail(sessions), [sessions]);

  const load = () => {
    Promise.all([getAdminAccess(), getAdminSessions()])
      .then(([accessData, sessionData]) => {
        setAccess(accessData.items);
        setSessions(sessionData.items);
        setError(null);
      })
      .catch((err: unknown) => {
        if (isAdminAuthError(err)) {
          onAuthLost();
          return;
        }
        setError(errorText(err));
      });
  };

  useEffect(() => {
    load();
  }, []);

  const addAccess = async () => {
    if (!email.trim()) return;
    try {
      await addAdminAccess(email);
      setEmail("");
      load();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    }
  };

  const revokeAccess = async (targetEmail: string) => {
    try {
      await revokeAdminAccess(targetEmail);
      load();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    }
  };

  const revokeSession = async (session: AdminSession) => {
    try {
      await revokeAdminSession(session.id);
      load();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    }
  };

  const confirmDialog =
    confirmAction?.kind === "access"
      ? {
          title: "Удалить администратора?",
          text: `${confirmAction.email} потеряет доступ к админ-панели. Все активные админ-сессии этого email будут завершены.`,
          confirmText: "Удалить",
          onConfirm: () => {
            const emailToRevoke = confirmAction.email;
            setConfirmAction(null);
            void revokeAccess(emailToRevoke);
          },
        }
      : {
          title: "Разлогинить сессию?",
          text: confirmAction
            ? `${confirmAction.session.email} будет вынужден войти в админ-панель заново.`
            : "",
          confirmText: "Разлогинить",
          onConfirm: () => {
            if (!confirmAction || confirmAction.kind !== "session") return;
            const sessionToRevoke = confirmAction.session;
            setConfirmAction(null);
            void revokeSession(sessionToRevoke);
          },
        };

  return (
    <div className="access-layout">
      <ConfirmDialog
        open={confirmAction !== null}
        title={confirmDialog.title}
        text={confirmDialog.text}
        confirmText={confirmDialog.confirmText}
        tone="danger"
        onConfirm={confirmDialog.onConfirm}
        onCancel={() => setConfirmAction(null)}
      />

      {error && <p className="error-banner">{error}</p>}

      <section className="admin-panel">
        <div className="admin-panel-header">
          <div>
            <span>Администраторы</span>
            <h2>Доступ к панели</h2>
          </div>
          <strong className="admin-count-badge">{activeAccess.length}</strong>
        </div>

        <div className="admin-access-form">
          <input
            value={email}
            placeholder="email@goldencompass.kz"
            onChange={(event) => setEmail(event.target.value)}
          />
          <button type="button" onClick={() => void addAccess()}>
            Добавить
          </button>
        </div>

        <div className="access-list">
          {access.map((item) => {
            const session = sessionByEmail.get(item.email);
            const fullName = item.full_name ?? session?.full_name ?? null;
            const hasSession = item.has_session || session !== undefined;

            return (
              <div key={item.email} className={`access-row ${!item.is_active ? "access-row-muted" : ""}`}>
                <div className="access-person">
                  <strong>{item.email}</strong>
                  {fullName && <span>{fullName}</span>}
                  <StatusChip item={item} hasSession={hasSession} />
                </div>
                {item.is_active && (
                  <button type="button" onClick={() => setConfirmAction({ kind: "access", email: item.email })}>
                    Удалить
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </section>

      <section className="admin-panel">
        <div className="admin-panel-header">
          <div>
            <span>Сессии</span>
            <h2>Активные входы</h2>
          </div>
          <strong className="admin-count-badge">{activeSessions.length}</strong>
        </div>

        <div className="session-table-wrap">
          <table className="session-table">
            <thead>
              <tr>
                <th>Администратор</th>
                <th>Создана</th>
                <th>Истекает</th>
                <th>Статус</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {sessions.map((session) => (
                <tr key={session.id}>
                  <td>
                    <strong>{session.full_name}</strong>
                    <span>{session.email}</span>
                  </td>
                  <td>{formatDateTime(session.created_at)}</td>
                  <td>{formatDateTime(session.expires_at)}</td>
                  <td>{session.is_active ? "Активна" : "Отозвана"}</td>
                  <td>
                    {session.is_active && (
                      <button
                        className="session-revoke-button"
                        type="button"
                        aria-label={`Разлогинить ${session.email}`}
                        title="Разлогинить"
                        onClick={() => setConfirmAction({ kind: "session", session })}
                      >
                        <LogOut size={17} />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

function StatusChip({ item, hasSession }: { item: AdminAccess; hasSession: boolean }) {
  if (!item.is_active) {
    return <span className="access-status access-status-muted">Удален</span>;
  }

  if (!hasSession) {
    return <span className="access-status access-status-pending">Ожидает входа</span>;
  }

  return <span className="access-status access-status-active">Активен</span>;
}

function latestSessionByEmail(sessions: AdminSession[]): Map<string, AdminSession> {
  const map = new Map<string, AdminSession>();
  for (const session of sessions) {
    const key = session.email.toLowerCase();
    const current = map.get(key);
    if (!current || new Date(session.created_at) > new Date(current.created_at)) {
      map.set(key, session);
    }
  }

  return map;
}

function filterSuspiciousForUser(
  activity: AdminSuspiciousActivity | null,
  userId: string,
): EmployeeSuspiciousActivity {
  const deviceMatches = activity?.device_matches.filter((match) => match.event.user_id === userId) ?? [];
  const ipMatches = activity?.ip_matches.filter((match) => match.event.user_id === userId) ?? [];
  return {
    deviceMatches,
    ipMatches,
    total: deviceMatches.length + ipMatches.length,
  };
}

function suspiciousCountsByUser(activity: AdminSuspiciousActivity | null): Map<string, number> {
  const counts = new Map<string, number>();
  for (const match of activity?.device_matches ?? []) {
    counts.set(match.event.user_id, (counts.get(match.event.user_id) ?? 0) + 1);
  }
  for (const match of activity?.ip_matches ?? []) {
    counts.set(match.event.user_id, (counts.get(match.event.user_id) ?? 0) + 1);
  }
  return counts;
}

function employeeFilterCounts(employees: AdminEmployeeSummary[]): {
	late: number;
	early: number;
} {
  return employees.reduce(
    (acc, employee) => {
      if (employee.late_count > 0) acc.late += 1;
      if (employee.early_leave_count > 0) acc.early += 1;
      return acc;
    },
    { late: 0, early: 0 },
  );
}

function calendarCells(days: AttendanceDaySummary[]): Array<AttendanceDaySummary | null> {
  if (days.length === 0) return [];
  const first = new Date(`${days[0].date}T00:00:00`);
  const firstWeekday = first.getDay() || 7;
  const cells: Array<AttendanceDaySummary | null> = Array.from({ length: firstWeekday - 1 }, () => null);
  cells.push(...days);
  while (cells.length % 7 !== 0) cells.push(null);
  return cells;
}

function monthFromURL(): string | null {
  const params = new URLSearchParams(window.location.search);
  const month = params.get("month");
  return month && /^\d{4}-\d{2}$/.test(month) ? month : null;
}

function currentMonth(): string {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
}

function monthOptions(reports: AdminReportRun[]): Array<{ value: string; label: string }> {
  const values = new Set<string>([currentMonth()]);
  for (const report of reports) {
    values.add(report.period_start.slice(0, 7));
  }

  return [...values]
    .sort((left, right) => right.localeCompare(left))
    .map((item) => ({
      value: item,
      label: item === currentMonth() ? "Текущий месяц" : monthTitle(item),
    }));
}

function uniqueYears(options: Array<{ value: string }>): string[] {
  return [...new Set(options.map((option) => option.value.slice(0, 4)))];
}

function monthTitle(value: string): string {
  const date = new Date(`${value}-01T00:00:00`);
  return new Intl.DateTimeFormat("ru-RU", { month: "long", year: "numeric" }).format(date);
}

function formatDayLong(value: string): string {
  const date = new Date(`${value}T00:00:00`);
  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "long",
    weekday: "long",
  }).format(date);
}

function formatDayShort(value: string): string {
  const date = new Date(`${value}T00:00:00`);
  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "short",
  }).format(date);
}

function formatOnlyTime(value: string): string {
  return new Intl.DateTimeFormat("ru-RU", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

type TimePickerTarget = {
  userId: string;
  field: "check_in_at" | "check_out_at";
  value: string;
  defaultHour: string;
};

function TimePickerField({
  value,
  disabled,
  placeholder,
  onOpen,
}: {
  value: string;
  disabled: boolean;
  placeholder: string;
  onOpen: () => void;
}) {
  return (
    <button className="time-picker-field" type="button" disabled={disabled} onClick={onOpen}>
      {value || placeholder}
    </button>
  );
}

function TimePickerDialog({
  target,
  onSelect,
  onCancel,
}: {
  target: TimePickerTarget | null;
  onSelect: (value: string) => void;
  onCancel: () => void;
}) {
  const [hour, setHour] = useState("08");
  const [minute, setMinute] = useState("00");

  useEffect(() => {
    if (!target) return;
    const parsed = parseTimeValue(target.value) ?? parseTimeValue(`${target.defaultHour}:00`);
    setHour(parsed?.hour ?? target.defaultHour);
    setMinute(parsed?.minute ?? "00");
  }, [target]);

  if (!target) return null;

  return (
    <div className="time-picker-backdrop" role="presentation" onMouseDown={onCancel}>
      <div
        className="time-picker-dialog"
        role="dialog"
        aria-modal="true"
        aria-label="Выбор времени"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="time-picker-header">
          <span>Выбор времени</span>
          <strong>{`${Number(hour)}:${minute}`}</strong>
        </div>
        <div className="time-picker-columns">
          <div>
            <span>Часы</span>
            <div className="time-picker-grid time-picker-grid-hours">
              {hourOptions.map((option) => (
                <button
                  key={option.value}
                  className={option.value === hour ? "time-picker-selected" : ""}
                  type="button"
                  onClick={() => setHour(option.value)}
                >
                  {option.label}
                </button>
              ))}
            </div>
          </div>
          <div>
            <span>Минуты</span>
            <div className="time-picker-grid time-picker-grid-minutes">
              {minuteOptions.map((option) => (
                <button
                  key={option}
                  className={option === minute ? "time-picker-selected" : ""}
                  type="button"
                  onClick={() => setMinute(option)}
                >
                  {option}
                </button>
              ))}
            </div>
          </div>
        </div>
        <div className="time-picker-actions">
          <button type="button" onClick={onCancel}>
            Отмена
          </button>
          <button type="button" onClick={() => onSelect(`${hour}:${minute}`)}>
            Выбрать
          </button>
        </div>
      </div>
    </div>
  );
}

function toTimeDisplayValue(value: string | null): string {
  if (!value) return "";
  const date = new Date(value);
  return `${date.getHours()}:${String(date.getMinutes()).padStart(2, "0")}`;
}

function repairTimeValue(draftValue: string | undefined, persistedValue: string | null): string {
  return draftValue || toTimeDisplayValue(persistedValue);
}

const hourOptions = Array.from({ length: 24 }, (_, hour) => ({
  value: String(hour).padStart(2, "0"),
  label: String(hour),
}));

const minuteOptions = Array.from({ length: 60 }, (_, minute) => String(minute).padStart(2, "0"));

function parseTimeValue(value: string): { hour: string; minute: string } | null {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const match = /^([01]?\d|2[0-3]):([0-5]\d)$/.exec(trimmed);
  if (!match) return null;
  return {
    hour: match[1].padStart(2, "0"),
    minute: match[2],
  };
}

function normalizeTimeInput(value: string): string | undefined {
  const parsed = parseTimeValue(value);
  return parsed ? `${parsed.hour}:${parsed.minute}` : undefined;
}

function buildOutageRepairItems(
  draft: Record<string, { check_in_at: string; check_out_at: string }>,
  employees: Array<{ user_id: string; check_in_at: string | null }>,
): {
  items: AdminOutageRepairItem[];
  hasInvalid: boolean;
  hasCheckOutWithoutCheckIn: boolean;
} {
  const items: AdminOutageRepairItem[] = [];
  const existingCheckIns = new Map(employees.map((employee) => [employee.user_id, Boolean(employee.check_in_at)]));
  let hasInvalid = false;
  let hasCheckOutWithoutCheckIn = false;

  for (const [userId, value] of Object.entries(draft)) {
    const checkIn = normalizeTimeInput(value.check_in_at);
    const checkOut = normalizeTimeInput(value.check_out_at);
    if ((value.check_in_at && !checkIn) || (value.check_out_at && !checkOut)) {
      hasInvalid = true;
      continue;
    }
    if (checkOut && !checkIn && !existingCheckIns.get(userId)) {
      hasCheckOutWithoutCheckIn = true;
    }
    if (checkIn || checkOut) {
      items.push({
        user_id: userId,
        check_in_at: checkIn,
        check_out_at: checkOut,
      });
    }
  }

  return { items, hasInvalid, hasCheckOutWithoutCheckIn };
}

function formatDateTime(value: string): string {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function minutesToClock(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return `${hours}:${String(rest).padStart(2, "0")}`;
}

function minutesToHours(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return rest === 0 ? `${hours} ч` : `${hours} ч ${rest} мин`;
}

function shortId(value: string): string {
  return value.length > 18 ? `${value.slice(0, 8)}...${value.slice(-6)}` : value;
}

function eventTypeText(value: "check_in" | "check_out"): string {
  return value === "check_in" ? "приход" : "уход";
}

function dayDotClass(day: AttendanceDaySummary): string {
  if (day.impacted_by_outage) return "calendar-dot-outage";
  if (day.late_minutes > 0 || day.early_leave_minutes > 0) return "calendar-dot-issue";
  if (day.status === "in_progress") return "calendar-dot-progress";
  return "calendar-dot-complete";
}
