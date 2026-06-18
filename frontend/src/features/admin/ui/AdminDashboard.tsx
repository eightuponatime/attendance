import {
  AlertTriangle,
  ArrowLeft,
  CalendarDays,
  ChevronDown,
  ClipboardList,
  FileSpreadsheet,
  LogOut,
  Radio,
  RotateCcw,
  Search,
  ShieldCheck,
  Trash2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  addAdminAccess,
  approveAdminExplanation,
  downloadAdminExcelReport,
  getAdminEmployee,
  getAdminEmployees,
  getAdminAuditLogs,
  getAdminExplanations,
  getAdminAccess,
  getAdminReports,
  getAdminSessions,
  getAdminSuspiciousActivity,
  getAdminSystemOutages,
  getAdminOutageDay,
  isAdminAuthError,
  repairAdminOutage,
  rejectAdminExplanation,
  rollbackAdminExplanation,
  restoreAdminEmployeeDay,
  revokeAdminAccess,
  revokeAdminSession,
  voidAdminEmployeeDay,
} from "../api/adminApi";
import { errorText } from "../../../shared/api/errors";
import { ConfirmDialog } from "../../../shared/ui/ConfirmDialog";
import type {
  AdminAccess,
  AdminAuditDecisionSource,
  AdminAuditLog,
  AdminEmployeeMonthDetail,
  AdminEmployeesMonth,
  AdminEmployeeSummary,
  AdminExplanation,
  AdminExplanationDecision,
  AttendanceExplanationStatus,
  AdminReportRun,
  AdminSession,
  AdminSuspiciousActivity,
  AdminSuspiciousDeviceMatch,
  AdminSuspiciousIPMatch,
  AdminSystemOutage,
  AdminOutageDay,
  AttendanceDaySummary,
  AdminMe,
} from "../../../shared/types/api";

type EmployeeSuspiciousActivity = {
  deviceMatches: AdminSuspiciousDeviceMatch[];
  ipMatches: AdminSuspiciousIPMatch[];
  total: number;
};

type AdminPageTab = "employees" | "access" | "outages" | "explanations" | "audit";
type EmployeeFilter = "all" | "late" | "early";
const DEFAULT_WORKDAY_START = "08:00";
const DEFAULT_WORKDAY_END = "17:00";

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
  const [explanations, setExplanations] = useState<AdminExplanation[]>([]);
  const [pageTab, setPageTab] = useState<AdminPageTab>("employees");
  const [query, setQuery] = useState("");
  const [employeeFilter, setEmployeeFilter] = useState<EmployeeFilter>("all");
  const [mobileDetailOpen, setMobileDetailOpen] = useState(false);
  const [isDownloadingExcel, setIsDownloadingExcel] = useState(false);
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
      Promise.all([
        getAdminEmployees(month),
        getAdminSuspiciousActivity(month),
        getAdminSystemOutages(month),
      ])
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
    getAdminExplanations(month, "pending")
      .then((data) => setExplanations(data.items))
      .catch((err: unknown) => {
        if (isAdminAuthError(err)) {
          onAuthLost();
          return;
        }
        setExplanations([]);
      });
  }, [month, onAuthLost]);

  const outageVersion = useMemo(
    () => outages.map((outage) => `${outage.id}:${outage.affected_business_date ?? ""}:${outage.resolved_at ?? ""}`).join("|"),
    [outages],
  );

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
  }, [handleError, month, outageVersion, selectedUserId]);

  const suspiciousCounts = useMemo(() => suspiciousCountsByUser(suspicious), [suspicious]);
  const unresolvedOutageCount = useMemo(
    () => outages.filter((outage) => outage.impacts_work_hours && !outage.resolved_at).length,
    [outages],
  );
  const pendingExplanationCount = useMemo(
    () => explanations.filter((item) => item.status === "pending").length,
    [explanations],
  );
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

  const downloadExcel = async () => {
    setIsDownloadingExcel(true);
    try {
      await downloadAdminExcelReport(month);
      setError(null);
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    } finally {
      setIsDownloadingExcel(false);
    }
  };

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

            <button
              className="admin-excel-button"
              type="button"
              aria-label="Скачать Excel отчет"
              title="Скачать Excel отчет"
              disabled={isDownloadingExcel}
              onClick={() => void downloadExcel()}
            >
              <FileSpreadsheet size={18} />
              <span>{isDownloadingExcel ? "Готовим" : "Excel"}</span>
            </button>
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
          {unresolvedOutageCount > 0 && <span>{unresolvedOutageCount}</span>}
        </button>
        <button
          className={pageTab === "explanations" ? "admin-page-tab-active" : ""}
          type="button"
          onClick={() => setPageTab("explanations")}
        >
          Заявки
          {pendingExplanationCount > 0 && <span>{pendingExplanationCount}</span>}
        </button>
        <button
          className={pageTab === "access" ? "admin-page-tab-active" : ""}
          type="button"
          onClick={() => setPageTab("access")}
        >
          Управление доступом
        </button>
        <button
          className={pageTab === "audit" ? "admin-page-tab-active" : ""}
          type="button"
          onClick={() => setPageTab("audit")}
        >
          Аудит
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
              <EmployeeDetail
                employee={selectedEmployee}
                suspicious={suspicious}
                month={month}
                onEmployeeChange={setSelectedEmployee}
                onAuthLost={onAuthLost}
              />
            ) : (
              <section className="admin-panel">
                <p className="muted-text">Выберите сотрудника</p>
              </section>
            )}
          </section>
        </div>
      ) : pageTab === "outages" ? (
        <OutagesPanel
          month={month}
          outages={outages}
          onAuthLost={onAuthLost}
          onOutageUpdated={(updated) => {
            setOutages((current) => current.map((outage) => (outage.id === updated.id ? updated : outage)));
          }}
        />
      ) : pageTab === "explanations" ? (
        <ExplanationsPanel month={month} onPendingChange={setExplanations} onAuthLost={onAuthLost} />
      ) : pageTab === "audit" ? (
        <AuditPanel month={month} onAuthLost={onAuthLost} />
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
  month,
  onEmployeeChange,
  onAuthLost,
}: {
  employee: AdminEmployeeMonthDetail;
  suspicious: AdminSuspiciousActivity | null;
  month: string;
  onEmployeeChange: (employee: AdminEmployeeMonthDetail) => void;
  onAuthLost: () => void;
}) {
  const markedDays = employee.days.filter((day) => day.status !== "empty");
  const todayDay = employee.days.find((day) => day.date === localISODate(new Date()));
  const defaultDay = todayDay ?? [...markedDays].reverse()[0] ?? employee.days.find((day) => day.status !== "empty") ?? employee.days[0];
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

          {selectedDay && (
            <SelectedAdminDay
              employee={employee}
              day={selectedDay}
              month={month}
              onEmployeeChange={onEmployeeChange}
              onAuthLost={onAuthLost}
            />
          )}
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
            className={[
              "admin-calendar-day",
              day.date === selectedDate ? "admin-calendar-day-selected" : "",
            ].join(" ")}
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

function SelectedAdminDay({
  employee,
  day,
  month,
  onEmployeeChange,
  onAuthLost,
}: {
  employee: AdminEmployeeMonthDetail;
  day: AttendanceDaySummary;
  month: string;
  onEmployeeChange: (employee: AdminEmployeeMonthDetail) => void;
  onAuthLost: () => void;
}) {
  const [mode, setMode] = useState<"void" | "restore" | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const actionLabel = day.voided ? "Восстановить день" : "Аннулировать день";
  const reasonLabel = day.voided ? "Причина восстановления" : "Причина аннулирования";

  const submit = async () => {
    const normalizedReason = reason.trim();
    if (!normalizedReason) {
      setError("Укажите причину");
      return;
    }

    setBusy(true);
    try {
      if (day.voided) {
        await restoreAdminEmployeeDay(employee.user_id, day.date, normalizedReason);
      } else {
        await voidAdminEmployeeDay(employee.user_id, day.date, normalizedReason);
      }
      const refreshed = await getAdminEmployee(employee.user_id, month);
      onEmployeeChange(refreshed);
      setMode(null);
      setReason("");
      setError(null);
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={`admin-day-card ${day.voided ? "admin-day-card-voided" : ""}`}>
      <strong>{formatDayLong(day.date)}</strong>
      {day.voided && (
        <div className="admin-day-void-note">
          <span>День аннулирован</span>
          <b>{day.void_reason}</b>
          <em>
            {day.voided_by_admin ? `Администратор: ${day.voided_by_admin}` : "Администратор не указан"}
            {day.voided_at ? ` · ${formatDateTime(day.voided_at)}` : ""}
          </em>
        </div>
      )}
      {day.impacted_by_outage && (
        <div className="admin-day-outage-note">
          <span>Сбой сервера</span>
            <b>Администратор может восстановить отметки после проверки</b>
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
      <div className="admin-day-danger-zone">
        <button
          type="button"
          className={day.voided ? "admin-day-restore-button" : "admin-day-void-button"}
          onClick={() => {
            setMode(day.voided ? "restore" : "void");
            setReason("");
            setError(null);
          }}
        >
          {day.voided ? <RotateCcw size={16} /> : <Trash2 size={16} />}
          {actionLabel}
        </button>
      </div>
      {mode && (
        <div className="admin-day-override-form">
          <label>
            {reasonLabel}
            <textarea
              value={reason}
              rows={3}
              placeholder={day.voided ? "Например: отпуск отменен, данные снова учитываются" : "Например: сотрудник был в отпуске, день не должен участвовать в статистике"}
              onChange={(event) => setReason(event.target.value)}
            />
          </label>
          {error && <p className="outage-validation">{error}</p>}
          <p>
            {day.voided
              ? "День снова попадет в статистику, если в нем есть отметки прихода и ухода."
              : "Исходные отметки останутся в базе, но день не будет учитываться в статистике и подозрительной активности."}
          </p>
          <div>
            <button type="button" onClick={() => { setMode(null); setReason(""); setError(null); }}>
              Отмена
            </button>
            <button type="button" disabled={busy || !reason.trim()} onClick={() => void submit()}>
              {busy ? "Сохраняем..." : actionLabel}
            </button>
          </div>
        </div>
      )}
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
  month,
  outages,
  onAuthLost,
  onOutageUpdated,
}: {
  month: string;
  outages: AdminSystemOutage[];
  onAuthLost: () => void;
  onOutageUpdated: (outage: AdminSystemOutage) => void;
}) {
  const impacted = outages.filter((outage) => outage.impacts_work_hours);
  const [selected, setSelected] = useState<AdminOutageDay | null>(null);
  const [relatedRequests, setRelatedRequests] = useState<AdminExplanation[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loadingDetails, setLoadingDetails] = useState(false);

  const loadOutageDetails = async (outage: AdminSystemOutage) => {
    setLoadingDetails(true);
    try {
      const [data, explanationData] = await Promise.all([
        getAdminOutageDay(outage.id),
        getAdminExplanations(month, ""),
      ]);
      const affectedDates = outageAffectedDates(data.outage);
      setSelected(data);
      setRelatedRequests(
        explanationData.items.filter((item) => affectedDates.includes(item.business_date)),
      );
      setError(null);
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    } finally {
      setLoadingDetails(false);
    }
  };

  const openOutage = async (outage: AdminSystemOutage) => {
    if (!outage.impacts_work_hours) return;
    await loadOutageDetails(outage);
  };

  return (
    <div className="outage-page">
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
                      ? `Проверено${outage.resolved_by ? `: ${outage.resolved_by}` : ""}. Запись сохранена в архиве.`
                      : outage.impacts_work_hours
                      ? "Затронул рабочее окно 06:00-19:00. Данные за день могут быть неполными."
                      : "Не попал в рабочее окно."}
                  </p>
                </div>
                <em>{outage.resolved_at ? "Проверено" : "Требует проверки"}</em>
              </button>
            ))}
          </div>
        )}
      </section>

      {selected && (
        <OutageDetailsDialog
          data={selected}
          requests={relatedRequests}
          loading={loadingDetails}
          onClose={() => setSelected(null)}
          onAuthLost={onAuthLost}
          onOutageUpdated={onOutageUpdated}
          onReload={() => void loadOutageDetails(selected.outage)}
          setError={setError}
        />
      )}
    </div>
  );
}

function OutageDetailsDialog({
  data,
  requests,
  loading,
  onClose,
  onAuthLost,
  onOutageUpdated,
  onReload,
  setError,
}: {
  data: AdminOutageDay;
  requests: AdminExplanation[];
  loading: boolean;
  onClose: () => void;
  onAuthLost: () => void;
  onOutageUpdated: (outage: AdminSystemOutage) => void;
  onReload: () => void;
  setError: (value: string | null) => void;
}) {
  const affectedDates = useMemo(() => outageAffectedDates(data.outage), [data.outage]);
  const [selectedDates, setSelectedDates] = useState<string[]>(affectedDates);
  const [activeIssueTab, setActiveIssueTab] = useState<"requests" | "missing">("requests");
  const [selectedRequestUserId, setSelectedRequestUserId] = useState<string | null>(null);
  const [selectedRequestId, setSelectedRequestId] = useState<string | null>(null);
  const [selectedMissingUserId, setSelectedMissingUserId] = useState<string | null>(null);
  const [resolutionNote, setResolutionNote] = useState(data.outage.resolution_note ?? "");
  const [confirmCloseOpen, setConfirmCloseOpen] = useState(false);
  const [confirmAction, setConfirmAction] = useState<"approve" | "reject" | null>(null);
  const [missingReason, setMissingReason] = useState("");
  const [missingBusy, setMissingBusy] = useState(false);
  const [timePicker, setTimePicker] = useState<TimePickerTarget | null>(null);
  const [draft, setDraft] = useState<{ check_in_at: string; check_out_at: string; review_note: string }>({
    check_in_at: "",
    check_out_at: "",
    review_note: "",
  });
  const selectedResolved = Boolean(data.outage.resolved_at);

  useEffect(() => {
    setSelectedDates(affectedDates);
    setResolutionNote(data.outage.resolution_note ?? "");
  }, [affectedDates.join("|"), data.outage.id, data.outage.resolution_note]);

  const visibleRequests = useMemo(
    () => requests.filter((item) => selectedDates.includes(item.business_date)),
    [requests, selectedDates],
  );
  const groupedRequests = useMemo(() => groupExplanationsByUser(visibleRequests), [visibleRequests]);
  const requestStats = useMemo(() => explanationStats(visibleRequests), [visibleRequests]);
  const usersWithRequests = useMemo(() => new Set(visibleRequests.map((item) => item.user_id)), [visibleRequests]);
  const emptyWithoutRequest = useMemo(
    () => data.employees.filter((employee) => (
      !employee.check_in_at &&
      !employee.check_out_at &&
      !usersWithRequests.has(employee.user_id)
    )),
    [data.employees, usersWithRequests],
  );
  const selectedRequestGroup = groupedRequests.find((group) => group.userId === selectedRequestUserId) ?? groupedRequests[0] ?? null;
  const selectedRequest = selectedRequestGroup?.items.find((item) => item.id === selectedRequestId) ??
    selectedRequestGroup?.items.find((item) => item.status === "pending") ??
    selectedRequestGroup?.items[0] ??
    null;
  const selectedMissing = emptyWithoutRequest.find((item) => item.user_id === selectedMissingUserId) ?? null;
  const missingBusinessDate = data.outage.affected_business_date ?? selectedDates[0] ?? "";

  useEffect(() => {
    setSelectedRequestUserId((current) => {
      if (current && groupedRequests.some((group) => group.userId === current)) return current;
      return groupedRequests.find((group) => group.items.some((item) => item.status === "pending"))?.userId ??
        groupedRequests[0]?.userId ??
        null;
    });
  }, [groupedRequests]);

  useEffect(() => {
    if (!selectedRequestGroup) {
      setSelectedRequestId(null);
      return;
    }
    setSelectedRequestId((current) => {
      if (current && selectedRequestGroup.items.some((item) => item.id === current)) return current;
      return selectedRequestGroup.items.find((item) => item.status === "pending")?.id ?? selectedRequestGroup.items[0]?.id ?? null;
    });
  }, [selectedRequestGroup]);

  useEffect(() => {
    if (!selectedRequest) return;
    setDraft({
      check_in_at: "",
      check_out_at: "",
      review_note: selectedRequest.review_note ?? "",
    });
  }, [selectedRequest?.id]);

  const checkInValue = repairTimeValue(draft.check_in_at, selectedRequest?.check_in_at ?? null);
  const checkOutValue = repairTimeValue(draft.check_out_at, selectedRequest?.check_out_at ?? null);
  const hasCheckOutWithoutCheckIn = Boolean(normalizeTimeInput(draft.check_out_at)) && !Boolean(checkInValue);
  const missingRequiredRepair = selectedRequest
    ? explanationMissingRequiredRepair(selectedRequest, checkInValue, checkOutValue)
    : null;
  const repairPolicyWarning = selectedRequest
    ? explanationRepairPolicyWarning(selectedRequest, draft.check_in_at, draft.check_out_at)
    : null;
  const selectedIsVoidRequest = selectedRequest?.reason_type === "void_day_request";
  const canEditCheckIn = Boolean(selectedRequest && selectedRequest.status === "pending" && explanationCanRepairCheckIn(selectedRequest.reason_type));
  const canEditCheckOut = Boolean(selectedRequest && selectedRequest.status === "pending" && explanationCanRepairCheckOut(selectedRequest.reason_type));
  const canApprove = Boolean(selectedRequest) && !hasCheckOutWithoutCheckIn && !missingRequiredRepair && !repairPolicyWarning;
  const canReject = Boolean(selectedRequest);

  const toggleDate = (date: string) => {
    setSelectedDates((current) => {
      if (current.includes(date) && current.length > 1) {
        return current.filter((item) => item !== date);
      }
      if (current.includes(date)) return current;
      return [...current, date].sort();
    });
  };

  const closeOutage = async () => {
    try {
      const note = resolutionNote.trim() || "Сбой проверен. Посещаемость из вкладки сбоев не изменялась; спорные дни обрабатываются через заявки сотрудников.";
      await repairAdminOutage(data.outage.id, note, []);
      const refreshed = await getAdminOutageDay(data.outage.id);
      onOutageUpdated(refreshed.outage);
      setConfirmCloseOpen(false);
      setError(null);
      onReload();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    }
  };

  const applyDecision = async (decision: "approve" | "reject") => {
    if (!selectedRequest) return;
    const payload = explanationDecisionPayload(selectedRequest, draft);

    try {
      if (decision === "approve") {
        await approveAdminExplanation(selectedRequest.id, payload);
      } else {
        await rejectAdminExplanation(selectedRequest.id, { review_note: payload.review_note });
      }
      setConfirmAction(null);
      setError(null);
      onReload();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    }
  };

  const voidMissingDay = async () => {
    if (!selectedMissing || !missingBusinessDate || !missingReason.trim()) return;
    setMissingBusy(true);
    try {
      await voidAdminEmployeeDay(selectedMissing.user_id, missingBusinessDate, missingReason.trim());
      setMissingReason("");
      setSelectedMissingUserId(null);
      setError(null);
      onReload();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    } finally {
      setMissingBusy(false);
    }
  };

  return (
    <div className="outage-dialog-backdrop" role="presentation" onMouseDown={onClose}>
      <section
        className="outage-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="outage-dialog-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="outage-dialog-header">
          <div>
            <span>Разбор инцидента</span>
            <h2 id="outage-dialog-title">{data.outage.affected_business_date ?? formatDayShort(data.outage.started_at)}</h2>
            <p>{formatDateTime(data.outage.started_at)} - {formatDateTime(data.outage.ended_at)}</p>
          </div>
          <button type="button" aria-label="Закрыть" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <p className={`outage-warning ${selectedResolved ? "outage-warning-resolved" : ""}`}>
          {selectedResolved
            ? "Сбой уже помечен как проверенный. Решения по сотрудникам остаются в заявках и audit log."
            : "Сбой группирует связанные заявки и помогает разобрать их вместе. Посещаемость из этого окна меняется только через решение конкретной заявки."}
        </p>

        <div className="outage-detail-grid">
          <div>
            <span>Связанные заявки</span>
            <strong>{visibleRequests.length}</strong>
          </div>
          <div>
            <span>На рассмотрении</span>
            <strong>{requestStats.pending}</strong>
          </div>
          <div>
            <span>Пустых дней без заявки</span>
            <strong>{emptyWithoutRequest.length}</strong>
          </div>
          <div>
            <span>Статус сбоя</span>
            <strong>{selectedResolved ? "Проверен" : "Требует проверки"}</strong>
          </div>
        </div>

        <div className="outage-date-filter">
          <span>Даты сбоя</span>
          <div>
            {affectedDates.map((date) => (
              <label key={date}>
                <input
                  type="checkbox"
                  checked={selectedDates.includes(date)}
                  onChange={() => toggleDate(date)}
                />
                {formatDayShort(date)}
              </label>
            ))}
          </div>
        </div>

        <div className="outage-dialog-body">
          <section className="outage-dialog-list">
            <div className="outage-issue-tabs">
              <button
                type="button"
                className={activeIssueTab === "requests" ? "outage-issue-tab-active" : ""}
                onClick={() => setActiveIssueTab("requests")}
              >
                Заявки <span>{groupedRequests.length}</span>
              </button>
              <button
                type="button"
                className={activeIssueTab === "missing" ? "outage-issue-tab-active" : ""}
                onClick={() => setActiveIssueTab("missing")}
              >
                Без заявки <span>{emptyWithoutRequest.length}</span>
              </button>
            </div>

            {activeIssueTab === "requests" ? (
              loading ? (
                <p className="muted-text">Загрузка заявок...</p>
              ) : groupedRequests.length === 0 ? (
                <p className="muted-text">По выбранным датам заявок нет.</p>
              ) : (
                <div className="outage-compact-table">
                  {groupedRequests.map((group) => (
                    <button
                      key={group.userId}
                      className={`outage-group-row ${group.userId === selectedRequestGroup?.userId ? "outage-compact-row-active" : ""}`}
                      type="button"
                      onClick={() => {
                        setActiveIssueTab("requests");
                        setSelectedMissingUserId(null);
                        setSelectedRequestUserId(group.userId);
                      }}
                    >
                      <strong>{group.fullName}</strong>
                      <span>
                        {group.items.length} {requestCountLabel(group.items.length)}
                        {" · "}
                        {group.items.filter((item) => item.status === "pending").length} на рассмотрении
                      </span>
                      <small>{group.items.map((item) => formatDayShort(item.business_date)).join(", ")}</small>
                    </button>
                  ))}
                </div>
              )
            ) : emptyWithoutRequest.length === 0 ? (
              <p className="muted-text">Нет пустых дней без заявки.</p>
            ) : (
              <div className="outage-compact-table">
                {emptyWithoutRequest.map((employee) => (
                  <button
                    key={employee.user_id}
                    className={employee.user_id === selectedMissingUserId ? "outage-compact-row-active" : ""}
                    type="button"
                    onClick={() => {
                      setActiveIssueTab("missing");
                      setSelectedMissingUserId(employee.user_id);
                      setSelectedRequestUserId(null);
                      setSelectedRequestId(null);
                      setMissingReason("");
                    }}
                  >
                    <strong>{employee.full_name}</strong>
                    <span>0</span>
                    <em>0</em>
                    <small>{employee.email}</small>
                  </button>
                ))}
              </div>
            )}
          </section>

          <section className="outage-dialog-review">
            {selectedMissing ? (
              <>
                <div className="admin-panel-header">
                  <div>
                    <span>{formatDayLong(missingBusinessDate)}</span>
                    <h2>{selectedMissing.full_name}</h2>
                    <p>{selectedMissing.email}</p>
                  </div>
                </div>
                <div className="explanation-void-admin-note">
                  <strong>Нет заявки от сотрудника</strong>
                  <span>Можно аннулировать день, если администратор точно знает причину. Действие попадет в audit log.</span>
                </div>
                <label className="outage-note">
                  Причина аннулирования
                  <textarea
                    value={missingReason}
                    rows={3}
                    placeholder="Например: сотрудник был в отпуске, день не должен участвовать в статистике"
                    onChange={(event) => setMissingReason(event.target.value)}
                  />
                </label>
                <button
                  type="button"
                  className="outage-save-button"
                  disabled={missingBusy || !missingReason.trim()}
                  onClick={() => void voidMissingDay()}
                >
                  {missingBusy ? "Сохраняем..." : "Аннулировать день"}
                </button>
              </>
            ) : selectedRequest ? (
              <>
                <div className="admin-panel-header">
                  <div>
                    <span>Заявки сотрудника</span>
                    <h2>{selectedRequestGroup?.fullName ?? selectedRequest.full_name}</h2>
                    <p>
                      {selectedRequestGroup?.email ?? selectedRequest.email}
                      {selectedRequestGroup ? ` · ${selectedRequestGroup.items.length} ${requestCountLabel(selectedRequestGroup.items.length)}` : ""}
                    </p>
                  </div>
                  <ExplanationAdminStatus status={selectedRequest.status} />
                </div>

                {selectedRequestGroup && (
                  <div className="outage-request-group-detail">
                    <div>
                      <strong>Дни заявки</strong>
                      <span>{selectedRequestGroup.items.map((item) => formatDayShort(item.business_date)).join(", ")}</span>
                    </div>
                    <div className="outage-request-days-table">
                      <div>
                        <span>Дата</span>
                        <span>Тип</span>
                        <span>Статус</span>
                      </div>
                      {selectedRequestGroup.items.map((item) => (
                        <button
                          key={item.id}
                          type="button"
                          className={item.id === selectedRequest.id ? "outage-request-day-active" : ""}
                        onClick={() => setSelectedRequestId(item.id)}
                      >
                        <span>{formatDayShort(item.business_date)}</span>
                        <span className={`explanation-type-chip explanation-type-${item.reason_type}`}>{reasonText(item.reason_type)}</span>
                        <span className={`explanation-mini-status explanation-mini-status-${item.status}`}>{explanationStatusText(item.status)}</span>
                      </button>
                    ))}
                  </div>
                  </div>
                )}

                <div className="explanation-review-card">
                  <span>Выбрана дата: {formatDayLong(selectedRequest.business_date)} · {reasonText(selectedRequest.reason_type)}</span>
                  <p>{selectedRequest.comment}</p>
                </div>

                <div className="admin-day-card explanation-current-day">
                  <div>
                    <span>Приход сейчас</span>
                    <b>{selectedRequest.check_in_at ? formatOnlyTime(selectedRequest.check_in_at) : "Нет"}</b>
                  </div>
                  <div>
                    <span>Уход сейчас</span>
                    <b>{selectedRequest.check_out_at ? formatOnlyTime(selectedRequest.check_out_at) : "Нет"}</b>
                  </div>
                </div>

                {selectedIsVoidRequest ? (
                  <div className="explanation-void-admin-note">
                    <strong>При одобрении день будет аннулирован</strong>
                    <span>Исходные отметки останутся в базе, но день перестанет участвовать в статистике и проверке подозрительной активности.</span>
                  </div>
                ) : (
                  <div className="outage-repair-row explanation-time-row">
                    <div>
                      <strong>Корректировка времени</strong>
                      <span>{explanationRepairHint(selectedRequest)}</span>
                    </div>
                    <label>
                      Приход
                      <TimePickerField
                        value={checkInValue}
                        disabled={!canEditCheckIn}
                        placeholder="8:00"
                        onOpen={() => setTimePicker({
                          userId: selectedRequest.id,
                          field: "check_in_at",
                          businessDate: selectedRequest.business_date,
                          value: checkInValue,
                          defaultHour: "08",
                          maxTime: selectedRequest.business_date === localISODate(new Date()) ? currentClockValue() : undefined,
                        })}
                      />
                    </label>
                    <label>
                      Уход
                      <TimePickerField
                        value={checkOutValue}
                        disabled={!canEditCheckOut}
                        placeholder="17:00"
                        onOpen={() => setTimePicker({
                          userId: selectedRequest.id,
                          field: "check_out_at",
                          businessDate: selectedRequest.business_date,
                          value: checkOutValue,
                          defaultHour: "17",
                          maxTime: selectedRequest.business_date === localISODate(new Date()) ? currentClockValue() : undefined,
                        })}
                      />
                    </label>
                  </div>
                )}

                {hasCheckOutWithoutCheckIn && <p className="outage-validation">Нельзя сохранить уход без прихода.</p>}
                {missingRequiredRepair && <p className="outage-validation">{missingRequiredRepair}</p>}
                {repairPolicyWarning && <p className="outage-validation">{repairPolicyWarning}</p>}

                <label className="outage-note">
                  Ответ администратора
                  <textarea
                    value={draft.review_note}
                    rows={3}
                    disabled={selectedRequest.status !== "pending"}
                    placeholder="Необязательно. Комментарий виден сотруднику"
                    onChange={(event) => setDraft((current) => ({ ...current, review_note: event.target.value }))}
                  />
                </label>

                {selectedRequest.status === "pending" ? (
                  <div className="explanation-review-actions">
                    <button type="button" disabled={!canReject} onClick={() => setConfirmAction("reject")}>
                      Отклонить
                    </button>
                    <button type="button" disabled={!canApprove} onClick={() => setConfirmAction("approve")}>
                      {selectedIsVoidRequest ? "Аннулировать день" : "Одобрить"}
                    </button>
                  </div>
                ) : (
                  <p className="muted-text">
                    Рассмотрено {selectedRequest.reviewed_by_admin_email ? selectedRequest.reviewed_by_admin_email : "администратором"}
                    {selectedRequest.reviewed_at ? ` · ${formatDateTime(selectedRequest.reviewed_at)}` : ""}
                  </p>
                )}
              </>
            ) : (
              <p className="muted-text">Выберите заявку слева или дождитесь обращений сотрудников.</p>
            )}
          </section>
        </div>

        <div className="outage-dialog-footer">
          {!selectedResolved ? (
            <>
              <label className="outage-note">
                Комментарий к проверке сбоя
                <textarea
                  value={resolutionNote}
                  rows={2}
                  placeholder="Необязательно. Например: связанные заявки обработаны отдельно"
                  onChange={(event) => setResolutionNote(event.target.value)}
                />
              </label>
              <button type="button" className="outage-save-button" onClick={() => setConfirmCloseOpen(true)}>
                Пометить сбой проверенным
              </button>
            </>
          ) : (
            <div className="outage-resolved-note">
              <strong>Проверено{data.outage.resolved_by ? `: ${data.outage.resolved_by}` : ""}</strong>
              {data.outage.resolution_note && <span>{data.outage.resolution_note}</span>}
            </div>
          )}
        </div>
      </section>

      <TimePickerDialog
        target={timePicker}
        onCancel={() => setTimePicker(null)}
        onSelect={(value) => {
          if (!timePicker) return;
          setDraft((current) => ({ ...current, [timePicker.field]: value }));
          setTimePicker(null);
        }}
      />
      <ConfirmDialog
        open={confirmAction !== null}
        title={confirmAction === "approve" ? (selectedIsVoidRequest ? "Аннулировать день?" : "Одобрить заявку?") : "Отклонить заявку?"}
        text={
          confirmAction === "approve"
            ? selectedIsVoidRequest
              ? "День будет исключен из статистики. Исходные отметки останутся в базе, действие попадет в audit log."
              : approvalConfirmText(selectedRequest, draft)
            : draft.review_note.trim()
              ? "Сотрудник увидит статус отклонения и комментарий администратора."
              : "Сотрудник увидит статус отклонения без комментария администратора."
        }
        confirmText={confirmAction === "approve" ? (selectedIsVoidRequest ? "Аннулировать" : "Одобрить") : "Отклонить"}
        tone={confirmAction === "reject" ? "danger" : "neutral"}
        onConfirm={() => {
          if (!confirmAction) return;
          void applyDecision(confirmAction);
        }}
        onCancel={() => setConfirmAction(null)}
      />
      <ConfirmDialog
        open={confirmCloseOpen}
        title="Пометить сбой проверенным?"
        text="Посещаемость сотрудников не изменится. Сбой уйдет в архив, а заявки сотрудников останутся в истории решений."
        confirmText="Пометить проверенным"
        tone="neutral"
        onConfirm={() => void closeOutage()}
        onCancel={() => setConfirmCloseOpen(false)}
      />
    </div>
  );
}

function ExplanationsPanel({
  month,
  onPendingChange,
  onAuthLost,
}: {
  month: string;
  onPendingChange: (items: AdminExplanation[]) => void;
  onAuthLost: () => void;
}) {
  const [items, setItems] = useState<AdminExplanation[]>([]);
  const [status, setStatus] = useState<AttendanceExplanationStatus | "all">("pending");
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [reviewSheetOpen, setReviewSheetOpen] = useState(false);
  const [draft, setDraft] = useState<{ check_in_at: string; check_out_at: string; review_note: string }>({
    check_in_at: "",
    check_out_at: "",
    review_note: "",
  });
  const [timePicker, setTimePicker] = useState<TimePickerTarget | null>(null);
  const [confirmAction, setConfirmAction] = useState<"approve" | "reject" | null>(null);
  const [confirmRejectAllOpen, setConfirmRejectAllOpen] = useState(false);
  const [bulkRejectNote, setBulkRejectNote] = useState("");
  const [bulkRejecting, setBulkRejecting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const data = await getAdminExplanations(month, status === "all" ? "" : status);
      setItems(data.items);
      onPendingChange(data.items.filter((item) => item.status === "pending"));
      setError(null);
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    }
  }, [month, onAuthLost, onPendingChange, status]);

  useEffect(() => {
    void load();
  }, [load]);

  const groupedItems = useMemo(() => groupExplanationsByUser(items), [items]);
  const selectedGroup = groupedItems.find((group) => group.userId === selectedUserId) ?? groupedItems[0] ?? null;
  const selected = selectedGroup?.items.find((item) => item.id === selectedId) ??
    selectedGroup?.items.find((item) => item.status === "pending") ??
    selectedGroup?.items[0] ??
    null;
  const pendingItems = items.filter((item) => item.status === "pending");

  useEffect(() => {
    setSelectedUserId((current) => {
      if (current && groupedItems.some((group) => group.userId === current)) return current;
      return groupedItems.find((group) => group.items.some((item) => item.status === "pending"))?.userId ??
        groupedItems[0]?.userId ??
        null;
    });
  }, [groupedItems]);

  useEffect(() => {
    if (!selectedGroup) {
      setSelectedId(null);
      return;
    }
    setSelectedId((current) => {
      if (current && selectedGroup.items.some((item) => item.id === current)) return current;
      return selectedGroup.items.find((item) => item.status === "pending")?.id ?? selectedGroup.items[0]?.id ?? null;
    });
  }, [selectedGroup]);

  useEffect(() => {
    if (!selected) return;
    setDraft({
      check_in_at: "",
      check_out_at: "",
      review_note: selected.review_note ?? "",
    });
  }, [selected?.id]);

  const checkInValue = repairTimeValue(draft.check_in_at, selected?.check_in_at ?? null);
  const checkOutValue = repairTimeValue(draft.check_out_at, selected?.check_out_at ?? null);
  const hasCheckOutWithoutCheckIn = Boolean(normalizeTimeInput(draft.check_out_at)) && !Boolean(checkInValue);
  const missingRequiredRepair = selected ? explanationMissingRequiredRepair(selected, checkInValue, checkOutValue) : null;
  const repairPolicyWarning = selected ? explanationRepairPolicyWarning(selected, draft.check_in_at, draft.check_out_at) : null;
  const selectedIsVoidRequest = selected?.reason_type === "void_day_request";
  const repairHint = selected ? explanationRepairHint(selected) : "Заполняйте только поля, которые нужно изменить.";
  const canEditCheckIn = Boolean(selected && selected.status === "pending" && explanationCanRepairCheckIn(selected.reason_type));
  const canEditCheckOut = Boolean(selected && selected.status === "pending" && explanationCanRepairCheckOut(selected.reason_type));
  const canApprove = Boolean(selected) && !hasCheckOutWithoutCheckIn && !missingRequiredRepair && !repairPolicyWarning;
  const canReject = Boolean(selected);

  const applyDecision = async (decision: "approve" | "reject") => {
    if (!selected) return;
    const payload = explanationDecisionPayload(selected, draft);

    try {
      if (decision === "approve") {
        await approveAdminExplanation(selected.id, payload);
      } else {
        await rejectAdminExplanation(selected.id, { review_note: payload.review_note });
      }
      setConfirmAction(null);
      await load();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    }
  };

  const rejectAll = async () => {
    const note = bulkRejectNote.trim();
    if (pendingItems.length === 0) return;
    setBulkRejecting(true);
    try {
      await Promise.all(
        pendingItems.map((item) => rejectAdminExplanation(item.id, { review_note: note || undefined })),
      );
      setBulkRejectNote("");
      setConfirmRejectAllOpen(false);
      await load();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    } finally {
      setBulkRejecting(false);
    }
  };

  return (
    <div className="explanations-layout">
      <section className="admin-panel explanations-list-panel">
        <div className="admin-panel-header">
          <div>
            <span>Пересмотр отметок</span>
            <h2>Заявки сотрудников</h2>
            <p>Объяснения по опозданиям, ранним уходам и пропущенным отметкам.</p>
          </div>
          <strong className="admin-count-badge">{items.filter((item) => item.status === "pending").length}</strong>
        </div>

        <div className="employee-filters">
          {([
            ["pending", "На рассмотрении"],
            ["approved", "Одобрено"],
            ["rejected", "Отклонено"],
            ["all", "Все"],
          ] as Array<[AttendanceExplanationStatus | "all", string]>).map(([value, label]) => (
            <button
              key={value}
              className={status === value ? "employee-filter-active" : ""}
              type="button"
              onClick={() => setStatus(value)}
            >
              {label}
            </button>
          ))}
        </div>

        {pendingItems.length > 0 && (
          <div className="explanation-bulk-reject">
            <label>
              <span>Ответ для массового отклонения</span>
              <textarea
                value={bulkRejectNote}
                rows={2}
                placeholder="Необязательно. Комментарий увидят сотрудники"
                onChange={(event) => setBulkRejectNote(event.target.value)}
              />
            </label>
            <button
              type="button"
              disabled={bulkRejecting}
              onClick={() => setConfirmRejectAllOpen(true)}
            >
              {bulkRejecting ? "Отклоняем..." : `Отклонить все (${pendingItems.length})`}
            </button>
          </div>
        )}

        {error && <p className="error-banner">{error}</p>}

        {groupedItems.length === 0 ? (
          <p className="muted-text">Заявок за выбранный период нет</p>
        ) : (
          <div className="explanations-list">
            {groupedItems.map((group) => (
              <button
                key={group.userId}
                className={`explanation-admin-row ${group.userId === selectedGroup?.userId ? "explanation-admin-row-active" : ""}`}
                type="button"
                onClick={() => {
                  setSelectedUserId(group.userId);
                  setSelectedId(group.items.find((item) => item.status === "pending")?.id ?? group.items[0].id);
                  setReviewSheetOpen(true);
                }}
              >
                <div>
                  <strong>{group.fullName}</strong>
                  <span>{group.email}</span>
                  <p>{group.items.map((item) => formatDayShort(item.business_date)).join(", ")}</p>
                </div>
                <em className="admin-row-count">{group.items.length}</em>
              </button>
            ))}
          </div>
        )}
      </section>

      {selected && (
        <>
          {reviewSheetOpen && (
            <button
              type="button"
              className="explanation-review-backdrop"
              aria-label="Закрыть заявку"
              onClick={() => setReviewSheetOpen(false)}
            />
          )}
          <section className={`admin-panel explanation-review-panel ${reviewSheetOpen ? "explanation-review-panel-open" : ""}`}>
            <button
              type="button"
              className="explanation-review-close"
              aria-label="Закрыть заявку"
              onClick={() => setReviewSheetOpen(false)}
            >
              <X size={20} />
            </button>
            <div className="admin-panel-header">
              <div>
                <span>Заявки сотрудника</span>
                <h2>{selectedGroup?.fullName ?? selected.full_name}</h2>
                <p>{selectedGroup?.email ?? selected.email}</p>
              </div>
              <ExplanationAdminStatus status={selected.status} />
            </div>

            {selectedGroup && (
              <div className="outage-request-days-table">
                <div>
                  <span>Дата</span>
                  <span>Тип</span>
                  <span>Статус</span>
                </div>
                {selectedGroup.items.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    className={item.id === selected.id ? "outage-request-day-active" : ""}
                    onClick={() => setSelectedId(item.id)}
                  >
                    <span>{formatDayShort(item.business_date)}</span>
                    <span className={`explanation-type-chip explanation-type-${item.reason_type}`}>{reasonText(item.reason_type)}</span>
                    <span className={`explanation-mini-status explanation-mini-status-${item.status}`}>{explanationStatusText(item.status)}</span>
                  </button>
                ))}
              </div>
            )}

            <div className="explanation-review-card">
              <span>{formatDayLong(selected.business_date)} · {reasonText(selected.reason_type)}</span>
              <p>{selected.comment}</p>
            </div>

            <div className="admin-day-card explanation-current-day">
              <div>
                <span>Приход сейчас</span>
                <b>{selected.check_in_at ? formatOnlyTime(selected.check_in_at) : "Нет"}</b>
              </div>
              <div>
                <span>Уход сейчас</span>
                <b>{selected.check_out_at ? formatOnlyTime(selected.check_out_at) : "Нет"}</b>
              </div>
            </div>

            {selectedIsVoidRequest ? (
              <div className="explanation-void-admin-note">
                <strong>При одобрении день будет аннулирован</strong>
                <span>Исходные отметки останутся в базе, но день перестанет участвовать в статистике и проверке подозрительной активности.</span>
              </div>
            ) : (
              <div className="outage-repair-row explanation-time-row">
                <div>
                  <strong>Корректировка времени</strong>
                  <span>{repairHint}</span>
                </div>
                <label>
                  Приход
                  <TimePickerField
                    value={checkInValue}
                    disabled={!canEditCheckIn}
                    placeholder="8:00"
                    onOpen={() => setTimePicker({
                      userId: selected.id,
                      field: "check_in_at",
                      businessDate: selected.business_date,
                      value: checkInValue,
                      defaultHour: "08",
                      maxTime: selected.business_date === localISODate(new Date()) ? currentClockValue() : undefined,
                    })}
                  />
                </label>
                <label>
                  Уход
                  <TimePickerField
                    value={checkOutValue}
                    disabled={!canEditCheckOut}
                    placeholder="17:00"
                    onOpen={() => setTimePicker({
                      userId: selected.id,
                      field: "check_out_at",
                      businessDate: selected.business_date,
                      value: checkOutValue,
                      defaultHour: "17",
                      maxTime: selected.business_date === localISODate(new Date()) ? currentClockValue() : undefined,
                    })}
                  />
                </label>
              </div>
            )}

            {hasCheckOutWithoutCheckIn && (
              <p className="outage-validation">Нельзя сохранить уход без прихода.</p>
            )}
            {missingRequiredRepair && (
              <p className="outage-validation">{missingRequiredRepair}</p>
            )}
            {repairPolicyWarning && (
              <p className="outage-validation">{repairPolicyWarning}</p>
            )}

            <label className="outage-note">
              Ответ администратора
              <textarea
                value={draft.review_note}
                rows={3}
                disabled={selected.status !== "pending"}
                placeholder="Необязательно. Комментарий виден сотруднику"
                onChange={(event) => setDraft((current) => ({ ...current, review_note: event.target.value }))}
              />
            </label>

            {selected.status === "pending" ? (
              <div className="explanation-review-actions">
                <button type="button" disabled={!canReject} onClick={() => setConfirmAction("reject")}>
                  Отклонить
                </button>
                <button type="button" disabled={!canApprove} onClick={() => setConfirmAction("approve")}>
                  {selectedIsVoidRequest ? "Аннулировать день" : "Одобрить"}
                </button>
              </div>
            ) : (
              <p className="muted-text">
                Рассмотрено {selected.reviewed_by_admin_email ? selected.reviewed_by_admin_email : "администратором"}
                {selected.reviewed_at ? ` · ${formatDateTime(selected.reviewed_at)}` : ""}
              </p>
            )}
          </section>
        </>
      )}

      <TimePickerDialog
        target={timePicker}
        onCancel={() => setTimePicker(null)}
        onSelect={(value) => {
          if (!timePicker) return;
          setDraft((current) => ({ ...current, [timePicker.field]: value }));
          setTimePicker(null);
        }}
      />
      <ConfirmDialog
        open={confirmAction !== null}
        title={confirmAction === "approve" ? (selectedIsVoidRequest ? "Аннулировать день?" : "Одобрить заявку?") : "Отклонить заявку?"}
        text={
          confirmAction === "approve"
            ? selectedIsVoidRequest
              ? "День будет исключен из статистики. Исходные отметки останутся в базе, действие попадет в audit log."
              : approvalConfirmText(selected, draft)
            : draft.review_note.trim()
              ? "Сотрудник увидит статус отклонения и комментарий администратора."
              : "Сотрудник увидит статус отклонения без комментария администратора."
        }
        confirmText={confirmAction === "approve" ? (selectedIsVoidRequest ? "Аннулировать" : "Одобрить") : "Отклонить"}
        tone={confirmAction === "reject" ? "danger" : "neutral"}
        onConfirm={() => {
          if (!confirmAction) return;
          void applyDecision(confirmAction);
        }}
        onCancel={() => setConfirmAction(null)}
      />
      <ConfirmDialog
        open={confirmRejectAllOpen}
        title="Отклонить все заявки?"
        text={
          bulkRejectNote.trim()
            ? `Будут отклонены все заявки в текущем списке: ${pendingItems.length}. Сотрудники увидят общий комментарий администратора.`
            : `Будут отклонены все заявки в текущем списке: ${pendingItems.length}. Комментарий сотрудникам не отправится.`
        }
        confirmText="Отклонить все"
        tone="danger"
        onConfirm={() => void rejectAll()}
        onCancel={() => setConfirmRejectAllOpen(false)}
      />
    </div>
  );
}

function ExplanationAdminStatus({ status }: { status: AttendanceExplanationStatus }) {
  return <span className={`explanation-status explanation-status-${status}`}>{explanationStatusText(status)}</span>;
}

function AuditPanel({ month, onAuthLost }: { month: string; onAuthLost: () => void }) {
  const [items, setItems] = useState<AdminAuditLog[]>([]);
  const [sourceFilter, setSourceFilter] = useState<AdminAuditDecisionSource | "all">("all");
  const [query, setQuery] = useState("");
  const [selectedRollback, setSelectedRollback] = useState<AdminAuditLog | null>(null);
  const [busyRollback, setBusyRollback] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadAuditLogs = useCallback(() => {
    getAdminAuditLogs(month)
      .then((data) => {
        setItems(data.items);
        setError(null);
      })
      .catch((err: unknown) => {
        if (isAdminAuthError(err)) {
          onAuthLost();
          return;
        }
        setError(errorText(err));
      });
  }, [month, onAuthLost]);

  useEffect(() => {
    loadAuditLogs();
  }, [loadAuditLogs]);

  const rollbackReview = async () => {
    if (!selectedRollback?.explanation_id) return;
    setBusyRollback(true);
    try {
      await rollbackAdminExplanation(selectedRollback.explanation_id);
      setSelectedRollback(null);
      loadAuditLogs();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        onAuthLost();
        return;
      }
      setError(errorText(err));
    } finally {
      setBusyRollback(false);
    }
  };

  const returnableAuditIds = useMemo(() => {
    const lastRollbackAt = new Map<string, number>();
    for (const item of items) {
      if (item.explanation_id && item.action === "explanation_rollback") {
        const createdAt = new Date(item.created_at).getTime();
        const current = lastRollbackAt.get(item.explanation_id) ?? 0;
        if (createdAt > current) {
          lastRollbackAt.set(item.explanation_id, createdAt);
        }
      }
    }

    const explanationWithActiveDecision = new Set<string>();
    for (const item of items) {
      if (
        !item.explanation_id ||
        !["explanation_approved", "explanation_rejected"].includes(item.action)
      ) {
        continue;
      }

      if (new Date(item.created_at).getTime() > (lastRollbackAt.get(item.explanation_id) ?? 0)) {
        explanationWithActiveDecision.add(item.explanation_id);
      }
    }

    const result = new Set<string>();
    for (const item of items) {
      if (
        item.explanation_id &&
        explanationWithActiveDecision.has(item.explanation_id) &&
        new Date(item.created_at).getTime() > (lastRollbackAt.get(item.explanation_id) ?? 0) &&
        ["explanation_approved", "explanation_rejected", "check_in_changed", "check_out_changed", "day_voided"].includes(item.action)
      ) {
        result.add(item.id);
      }
    }

    return result;
  }, [items]);

  const canReturnToReview = (item: AdminAuditLog) => Boolean(
    item.explanation_id &&
    returnableAuditIds.has(item.id) &&
    ["explanation_approved", "explanation_rejected", "check_in_changed", "check_out_changed", "day_voided"].includes(item.action),
  );

  const visibleItems = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    return items.filter((item) => {
      if (sourceFilter !== "all" && item.decision_source !== sourceFilter) {
        return false;
      }
      if (!normalizedQuery) {
        return true;
      }
      return auditSearchText(item).includes(normalizedQuery);
    });
  }, [items, query, sourceFilter]);

  return (
    <section className="admin-panel audit-panel">
      <div className="admin-panel-header">
        <div>
          <span>Журнал действий</span>
          <h2>Аудит администраторов</h2>
          <p>Показываются действия за выбранный месяц.</p>
        </div>
        <strong className="admin-count-badge">{items.length}</strong>
      </div>

      <div className="audit-filters">
        {([
          { value: "all", label: "Все", count: items.length },
          {
            value: "employee_request",
            label: "По заявке сотрудника",
            count: items.filter((item) => item.decision_source === "employee_request").length,
          },
          {
            value: "admin_decision",
            label: "По решению администрации",
            count: items.filter((item) => item.decision_source === "admin_decision").length,
          },
        ] as const).map((filter) => (
          <button
            key={filter.value}
            type="button"
            className={sourceFilter === filter.value ? "audit-filter-active" : ""}
            onClick={() => setSourceFilter(filter.value)}
          >
            {filter.label}
            <span>{filter.count}</span>
          </button>
        ))}
      </div>

      <label className="admin-search audit-search">
        <Search size={16} />
        <input
          value={query}
          placeholder="Поиск по администратору, сотруднику, дате или действию"
          onChange={(event) => setQuery(event.target.value)}
        />
      </label>
      <p className="audit-result-count">
        Найдено: {visibleItems.length} из {items.length}
      </p>

      {error && <p className="error-banner">{error}</p>}

      {visibleItems.length === 0 ? (
        <p className="muted-text">Действий за выбранный период нет</p>
      ) : (
        <div className="audit-list">
          {visibleItems.map((item) => (
            <article key={item.id} className={`audit-row audit-row-${item.action}`}>
              <div className={`audit-icon audit-icon-${item.action}`}>
                <ClipboardList size={18} />
              </div>
              <div>
                <strong className={`audit-action-chip audit-action-${item.action}`}>{auditActionText(item.action)}</strong>
                <small className={`audit-source audit-source-${item.decision_source}`}>
                  {auditDecisionSourceText(item.decision_source)}
                </small>
                <span>
                  {item.full_name ?? item.email ?? "Сотрудник не найден"}
                  {item.business_date ? ` · ${formatDayShort(item.business_date)}` : ""}
                </span>
                <p>{auditReasonText(item.reason)}</p>
                <em>{item.admin_email} · {formatDateTime(item.created_at)}</em>
                {canReturnToReview(item) && (
                  <button
                    type="button"
                    className="audit-rollback-button"
                    onClick={() => setSelectedRollback(item)}
                  >
                    Вернуть на рассмотрение
                  </button>
                )}
              </div>
              <AuditTimes item={item} />
            </article>
          ))}
        </div>
      )}
      <ConfirmDialog
        open={selectedRollback !== null}
        title="Вернуть заявку на рассмотрение?"
        text="Решение администратора будет отменено. Если при одобрении менялись приход, уход или день был аннулирован, данные вернутся к состоянию до решения. Действие попадет в audit log."
        confirmText={busyRollback ? "Возвращаем..." : "Вернуть на рассмотрение"}
        tone="danger"
        onConfirm={() => void rollbackReview()}
        onCancel={() => {
          if (!busyRollback) setSelectedRollback(null);
        }}
      />
    </section>
  );
}

function AuditTimes({ item }: { item: AdminAuditLog }) {
  const oldTimes = [item.old_check_in_at, item.old_check_out_at].filter(Boolean).map((value) => formatOnlyTime(value as string));
  const newTimes = [item.new_check_in_at, item.new_check_out_at].filter(Boolean).map((value) => formatOnlyTime(value as string));
  if (oldTimes.length === 0 && newTimes.length === 0) {
    return null;
  }

  return (
    <div className="audit-times">
      {oldTimes.length > 0 && <span>Было: {oldTimes.join(" - ")}</span>}
      {newTimes.length > 0 && <span>Стало: {newTimes.join(" - ")}</span>}
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
  businessDate: string;
  value: string;
  defaultHour: string;
  maxTime?: string;
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
    <button
      className={`time-picker-field ${value ? "" : "time-picker-field-empty"}`}
      type="button"
      disabled={disabled}
      onClick={onOpen}
    >
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
  const maxParsed = target?.maxTime ? parseTimeValue(target.maxTime) : null;

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
                  disabled={isClockAfter(option.value, "00", maxParsed)}
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
                  disabled={isClockAfter(hour, option, maxParsed)}
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
          <button type="button" disabled={isClockAfter(hour, minute, maxParsed)} onClick={() => onSelect(`${hour}:${minute}`)}>
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

function explanationMissingRequiredRepair(
  explanation: AdminExplanation,
  checkInValue: string,
  checkOutValue: string,
): string | null {
  if (explanation.reason_type === "void_day_request") {
    return null;
  }

  switch (explanation.reason_type) {
    case "late":
      return normalizeTimeInput(toTimeDisplayValue(explanation.check_in_at)) === normalizeTimeInput(checkInValue)
        ? `Для заявки по опозданию выберите новое время прихода не раньше ${formatPolicyTime(DEFAULT_WORKDAY_START)}.`
        : null;
    case "early_leave":
      return normalizeTimeInput(toTimeDisplayValue(explanation.check_out_at)) === normalizeTimeInput(checkOutValue)
        ? `Для заявки по раннему уходу выберите новое время ухода не позже ${formatPolicyTime(DEFAULT_WORKDAY_END)}.`
        : null;
    case "missing_day":
      if (!checkInValue && !checkOutValue) return "Для полностью пустого дня выберите приход и уход.";
      if (!checkInValue) return "Для полностью пустого дня выберите приход.";
      if (!checkOutValue) return "Для полностью пустого дня выберите уход.";
      return null;
    case "missing_check_in":
      return checkInValue ? null : "Для заявки без прихода выберите время прихода.";
    case "missing_check_out":
      return checkOutValue ? null : "Для заявки без ухода выберите время ухода.";
    default:
      return null;
  }
}

function explanationRepairHint(explanation: AdminExplanation): string {
  switch (explanation.reason_type) {
    case "late":
      return `Выберите только новое время прихода: не раньше ${formatPolicyTime(DEFAULT_WORKDAY_START)} и не то же самое, что уже записано. Уход по этой заявке не меняется.`;
    case "early_leave":
      return `Выберите только новое время ухода: не позже ${formatPolicyTime(DEFAULT_WORKDAY_END)} и не то же самое, что уже записано. Приход по этой заявке не меняется.`;
    case "missing_check_in":
      return "Выберите только время прихода, которое нужно добавить в посещаемость.";
    case "missing_check_out":
      return "Выберите только время ухода, которое нужно добавить в посещаемость.";
    case "missing_day":
      return "Выберите приход и уход, которые нужно добавить в посещаемость.";
    case "void_day_request":
      return "";
  }
}

function explanationCanRepairCheckIn(reasonType: AdminExplanation["reason_type"]): boolean {
  return ["late", "missing_check_in", "missing_day"].includes(reasonType);
}

function explanationCanRepairCheckOut(reasonType: AdminExplanation["reason_type"]): boolean {
  return ["early_leave", "missing_check_out", "missing_day"].includes(reasonType);
}

function explanationDecisionPayload(
  explanation: AdminExplanation,
  draft: { check_in_at: string; check_out_at: string; review_note: string },
): AdminExplanationDecision {
  return {
    review_note: draft.review_note.trim() || undefined,
    check_in_at: explanationCanRepairCheckIn(explanation.reason_type) ? normalizeTimeInput(draft.check_in_at) : undefined,
    check_out_at: explanationCanRepairCheckOut(explanation.reason_type) ? normalizeTimeInput(draft.check_out_at) : undefined,
  };
}

function explanationRepairPolicyWarning(
  explanation: AdminExplanation,
  draftCheckIn: string,
  draftCheckOut: string,
): string | null {
  const checkIn = normalizeTimeInput(draftCheckIn);
  const checkOut = normalizeTimeInput(draftCheckOut);

  if (explanation.reason_type === "late" && checkIn && compareClock(checkIn, DEFAULT_WORKDAY_START) < 0) {
    return `Для заявки по опозданию новое время прихода не может быть раньше ${formatPolicyTime(DEFAULT_WORKDAY_START)}.`;
  }
  if (explanation.reason_type === "early_leave" && checkOut && compareClock(checkOut, DEFAULT_WORKDAY_END) > 0) {
    return `Для заявки по раннему уходу новое время ухода не может быть позже ${formatPolicyTime(DEFAULT_WORKDAY_END)}.`;
  }

  return null;
}

function approvalConfirmText(explanation: AdminExplanation | null, draft: { check_in_at: string; check_out_at: string }): string {
  if (!explanation) {
    return "Заявка будет одобрена.";
  }

  const changes: string[] = [];
  const checkIn = normalizeTimeInput(draft.check_in_at);
  const checkOut = normalizeTimeInput(draft.check_out_at);
  if (checkIn) changes.push(`приход: ${formatPolicyTime(checkIn)}`);
  if (checkOut) changes.push(`уход: ${formatPolicyTime(checkOut)}`);

  if (changes.length === 0) {
    return "Время посещения не изменится. Будет одобрен только статус заявки сотрудника, действие попадет в audit log.";
  }

  return `Будет записано новое время: ${changes.join(", ")}. Изменение попадет в посещаемость и audit log.`;
}

function compareClock(left: string, right: string): number {
  return clockMinutes(left) - clockMinutes(right);
}

function clockMinutes(value: string): number {
  const parsed = parseTimeValue(value);
  if (!parsed) return 0;
  return Number(parsed.hour) * 60 + Number(parsed.minute);
}

function formatPolicyTime(value: string): string {
  const parsed = parseTimeValue(value);
  if (!parsed) return value;
  return `${Number(parsed.hour)}:${parsed.minute}`;
}

function isClockAfter(
  hour: string,
  minute: string,
  max: { hour: string; minute: string } | null,
): boolean {
  if (!max) return false;
  const valueMinutes = Number(hour) * 60 + Number(minute);
  const maxMinutes = Number(max.hour) * 60 + Number(max.minute);
  return valueMinutes > maxMinutes;
}

function currentClockValue(): string {
  const now = new Date();
  return `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`;
}

function localISODate(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
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

function explanationStats(items: AdminExplanation[]): Record<AttendanceExplanationStatus, number> {
  return {
    pending: items.filter((item) => item.status === "pending").length,
    approved: items.filter((item) => item.status === "approved").length,
    rejected: items.filter((item) => item.status === "rejected").length,
  };
}

function requestCountLabel(count: number): string {
  const lastTwo = count % 100;
  const last = count % 10;
  if (lastTwo >= 11 && lastTwo <= 14) return "заявок";
  if (last === 1) return "заявка";
  if (last >= 2 && last <= 4) return "заявки";
  return "заявок";
}

function groupExplanationsByUser(items: AdminExplanation[]): Array<{
  userId: string;
  fullName: string;
  email: string;
  items: AdminExplanation[];
}> {
  const groups = new Map<string, {
    userId: string;
    fullName: string;
    email: string;
    items: AdminExplanation[];
  }>();

  for (const item of items) {
    const group = groups.get(item.user_id) ?? {
      userId: item.user_id,
      fullName: item.full_name,
      email: item.email,
      items: [],
    };
    group.items.push(item);
    groups.set(item.user_id, group);
  }

  return [...groups.values()].map((group) => ({
    ...group,
    items: group.items.sort((left, right) => left.business_date.localeCompare(right.business_date)),
  }));
}

function outageAffectedDates(outage: AdminSystemOutage): string[] {
  const result = new Set<string>();
  if (outage.affected_business_date) {
    result.add(outage.affected_business_date);
  }

  const start = new Date(outage.started_at);
  const end = new Date(outage.ended_at);
  const cursor = new Date(start.getFullYear(), start.getMonth(), start.getDate());
  const last = new Date(end.getFullYear(), end.getMonth(), end.getDate());

  while (cursor <= last) {
    const weekday = cursor.getDay();
    if (weekday !== 0 && weekday !== 6) {
      result.add(localISODate(cursor));
    }
    cursor.setDate(cursor.getDate() + 1);
  }

  return [...result].sort();
}

function reasonText(value: AdminExplanation["reason_type"]): string {
  switch (value) {
    case "late":
      return "Опоздание";
    case "early_leave":
      return "Ранний уход";
    case "missing_check_in":
      return "Нет прихода";
    case "missing_check_out":
      return "Нет ухода";
    case "missing_day":
      return "Нет отметок";
    case "void_day_request":
      return "Исключить день";
  }
}

function explanationStatusText(value: AttendanceExplanationStatus): string {
  switch (value) {
    case "pending":
      return "На рассмотрении";
    case "approved":
      return "Одобрено";
    case "rejected":
      return "Отклонено";
  }
}

function auditActionText(value: AdminAuditLog["action"]): string {
  switch (value) {
    case "day_voided":
      return "День аннулирован";
    case "day_restored":
      return "День восстановлен";
    case "check_in_changed":
      return "Приход изменен";
    case "check_out_changed":
      return "Уход изменен";
    case "explanation_approved":
      return "Заявка одобрена";
    case "explanation_rejected":
      return "Заявка отклонена";
    case "explanation_rollback":
      return "Решение по заявке отменено";
    case "system_outage_resolved":
      return "Сбой сервера проверен";
  }
}

function auditReasonText(value: string): string {
  switch (value) {
    case "employee_explanation_approved":
      return "Исправлено по одобренной заявке сотрудника";
    case "employee_explanation_rejected":
      return "Заявка сотрудника отклонена";
    case "employee_explanation_rollback":
      return "Заявка возвращена на рассмотрение";
    case "system_outage_repair":
      return "Восстановлено после сбоя сервера";
    case "system_outage_resolved":
      return "Сбой сервера проверен администратором";
    default:
      return value;
  }
}

function auditDecisionSourceText(value: AdminAuditDecisionSource): string {
  switch (value) {
    case "employee_request":
      return "По заявке сотрудника";
    case "admin_decision":
      return "По решению администрации";
  }
}

function auditSearchText(item: AdminAuditLog): string {
  return [
    item.admin_email,
    item.email,
    item.full_name,
    item.business_date,
    item.created_at,
    auditActionText(item.action),
    auditReasonText(item.reason),
    auditDecisionSourceText(item.decision_source),
    item.old_check_in_at ? formatOnlyTime(item.old_check_in_at) : "",
    item.old_check_out_at ? formatOnlyTime(item.old_check_out_at) : "",
    item.new_check_in_at ? formatOnlyTime(item.new_check_in_at) : "",
    item.new_check_out_at ? formatOnlyTime(item.new_check_out_at) : "",
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
}

function dayDotClass(day: AttendanceDaySummary): string {
  if (day.voided) return "calendar-dot-muted";
  if (day.impacted_by_outage) return "calendar-dot-outage";
  if (day.explanations.some((item) => item.status === "pending")) return "calendar-dot-outage";
  if (isTodayInProgress(day)) return "calendar-dot-live";
  if (day.late_minutes > 0 || day.early_leave_minutes > 0) return "calendar-dot-issue";
  if (day.status === "in_progress") return "calendar-dot-progress";
  return "calendar-dot-complete";
}

function isTodayInProgress(day: AttendanceDaySummary): boolean {
  return day.date === localISODate(new Date()) && Boolean(day.check_in_at) && !day.check_out_at;
}
