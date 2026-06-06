import { ChevronLeft, ChevronRight } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getAttendanceSummary } from "../api/attendanceApi";
import { errorText } from "../../../shared/api/errors";
import { useI18n } from "../../../shared/i18n/i18n";
import type { AttendanceDaySummary, AttendanceSummary } from "../../../shared/types/api";
import { AttendanceExplanationBox } from "./AttendanceExplanationBox";

export function CalendarScreen() {
  const { formatDate, t, weekdayLabels } = useI18n();
  const [visibleMonth, setVisibleMonth] = useState(() => startOfMonth(new Date()));
  const [mode, setMode] = useState<"month" | "year">("month");
  const [summary, setSummary] = useState<AttendanceSummary | null>(null);
  const [selectedDate, setSelectedDate] = useState(() => formatISODate(new Date()));
  const [error, setError] = useState<string | null>(null);
  const touchStartX = useRef<number | null>(null);

  const range = useMemo(() => {
    const from = startOfMonth(visibleMonth);
    const to = endOfMonth(visibleMonth);
    return { from, to };
  }, [visibleMonth]);

  const loadSummary = useCallback(() => {
    getAttendanceSummary(formatISODate(range.from), formatISODate(range.to))
      .then((data) => {
        setSummary(data);
        setError(null);
      })
      .catch((err: unknown) => setError(errorText(err)));
  }, [range.from, range.to]);

  useEffect(() => {
    loadSummary();
  }, [loadSummary]);

  useEffect(() => {
    const refreshWhenVisible = () => {
      if (document.visibilityState === "visible") {
        loadSummary();
      }
    };
    window.addEventListener("focus", loadSummary);
    document.addEventListener("visibilitychange", refreshWhenVisible);
    return () => {
      window.removeEventListener("focus", loadSummary);
      document.removeEventListener("visibilitychange", refreshWhenVisible);
    };
  }, [loadSummary]);

  useEffect(() => {
    const selected = new Date(`${selectedDate}T00:00:00`);
    if (
      selected.getFullYear() !== visibleMonth.getFullYear() ||
      selected.getMonth() !== visibleMonth.getMonth()
    ) {
      setSelectedDate(formatISODate(visibleMonth));
    }
  }, [selectedDate, visibleMonth]);

  const daysByDate = useMemo(() => {
    const map = new Map<string, AttendanceDaySummary>();
    summary?.days.forEach((day) => map.set(day.date, day));
    return map;
  }, [summary]);

  const selectedDay = daysByDate.get(selectedDate) ?? emptyDay(selectedDate);
  const moveMonth = (direction: -1 | 1) => {
    setVisibleMonth((current) => addMonths(current, direction));
  };
  const moveYear = (direction: -1 | 1) => {
    setVisibleMonth((current) => new Date(current.getFullYear() + direction, current.getMonth(), 1));
  };

  return (
    <section className="screen-content">
      <div className="calendar-header">
        <button
          className="calendar-month-button"
          type="button"
          aria-label={mode === "year" ? t("calendar.previousYear") : t("calendar.previousMonth")}
          onClick={() => (mode === "year" ? moveYear(-1) : moveMonth(-1))}
        >
          <ChevronLeft size={24} />
        </button>

        <button
          className="calendar-title-button"
          type="button"
          onClick={() => setMode((current) => (current === "month" ? "year" : "month"))}
        >
          <span>{visibleMonth.getFullYear()}</span>
          <strong>{mode === "year" ? t("calendar.monthSelect") : monthTitle(visibleMonth, formatDate)}</strong>
        </button>

        <button
          className="calendar-month-button"
          type="button"
          aria-label={mode === "year" ? t("calendar.nextYear") : t("calendar.nextMonth")}
          onClick={() => (mode === "year" ? moveYear(1) : moveMonth(1))}
        >
          <ChevronRight size={24} />
        </button>
      </div>

      {error && <p className="error-banner">{error}</p>}

      {mode === "month" ? (
        <>
          <section
            className="calendar-card"
            onWheel={(event) => {
              if (Math.abs(event.deltaX) < 45 || Math.abs(event.deltaX) < Math.abs(event.deltaY)) {
                return;
              }
              moveMonth(event.deltaX > 0 ? 1 : -1);
            }}
            onTouchStart={(event) => {
              touchStartX.current = event.touches[0]?.clientX ?? null;
            }}
            onTouchEnd={(event) => {
              if (touchStartX.current === null) {
                return;
              }

              const delta = touchStartX.current - (event.changedTouches[0]?.clientX ?? touchStartX.current);
              touchStartX.current = null;
              if (Math.abs(delta) > 48) {
                moveMonth(delta > 0 ? 1 : -1);
              }
            }}
          >
            <div className="calendar-weekdays" aria-hidden="true">
              {weekdayLabels().map((day) => (
                <span key={day}>{day}</span>
              ))}
            </div>

            <MonthGrid
              month={visibleMonth}
              daysByDate={daysByDate}
              selectedDate={selectedDate}
              onSelectDate={setSelectedDate}
            />
          </section>

          <CalendarDayCard day={selectedDay} onSubmitted={loadSummary} />
        </>
      ) : (
        <YearSwitcher
          year={visibleMonth.getFullYear()}
          selectedMonth={visibleMonth.getMonth()}
          selectedDate={selectedDate}
          formatDate={formatDate}
          weekdayLabels={weekdayLabels()}
          onSelectMonth={(month) => {
            setVisibleMonth(new Date(visibleMonth.getFullYear(), month, 1));
            setMode("month");
          }}
        />
      )}
    </section>
  );
}

function MonthGrid({
  month,
  daysByDate,
  selectedDate,
  onSelectDate,
}: {
  month: Date;
  daysByDate: Map<string, AttendanceDaySummary>;
  selectedDate: string;
  onSelectDate: (date: string) => void;
}) {
  return (
    <div className="calendar-grid">
      {calendarCells(month).map((cell, index) => {
        if (!cell) {
          return <div key={`blank-${index}`} className="calendar-day-placeholder" />;
        }

        const date = formatISODate(cell.date);
        const day = daysByDate.get(date);
        const isSelected = date === selectedDate;
        const isToday = date === formatISODate(new Date());

        return (
          <button
            key={date}
            className={[
              "calendar-day",
              isSelected ? "calendar-day-selected" : "",
              isToday && !isSelected ? "calendar-day-today" : "",
            ].join(" ")}
            type="button"
            onClick={() => onSelectDate(date)}
          >
            <span>{cell.date.getDate()}</span>
            {day && (day.status !== "empty" || day.impacted_by_outage) && <i className={dayDotClass(day)} />}
          </button>
        );
      })}
    </div>
  );
}

function YearSwitcher({
  year,
  selectedMonth,
  selectedDate,
  formatDate,
  weekdayLabels,
  onSelectMonth,
}: {
  year: number;
  selectedMonth: number;
  selectedDate: string;
  formatDate: (date: Date, options: Intl.DateTimeFormatOptions) => string;
  weekdayLabels: string[];
  onSelectMonth: (month: number) => void;
}) {
  return (
    <section className="year-switcher">
      {Array.from({ length: 12 }, (_, month) => (
        <button
          key={month}
          className={`year-month ${month === selectedMonth ? "year-month-active" : ""}`}
          type="button"
          onClick={() => onSelectMonth(month)}
        >
          <strong>{monthTitle(new Date(year, month, 1), formatDate)}</strong>
          <MiniMonth year={year} month={month} selectedDate={selectedDate} weekdayLabels={weekdayLabels} />
        </button>
      ))}
    </section>
  );
}

function MiniMonth({
  year,
  month,
  selectedDate,
  weekdayLabels,
}: {
  year: number;
  month: number;
  selectedDate: string;
  weekdayLabels: string[];
}) {
  return (
    <div className="mini-month">
      {weekdayLabels.map((day) => (
        <span key={day} className="mini-weekday">
          {day}
        </span>
      ))}
      {calendarCells(new Date(year, month, 1)).map((cell, index) => {
        if (!cell) {
          return <span key={`mini-blank-${index}`} />;
        }

        const date = formatISODate(cell.date);
        return (
          <span key={date} className={date === selectedDate ? "mini-day-selected" : undefined}>
            {cell.date.getDate()}
          </span>
        );
      })}
    </div>
  );
}

function CalendarDayCard({ day, onSubmitted }: { day: AttendanceDaySummary; onSubmitted: () => void }) {
  const { formatDateString, locale, t } = useI18n();
  return (
    <section className="stats-card selected-day-card">
      <div className="selected-day-header">
        <strong>{formatDayLong(day.date, formatDateString)}</strong>
        <span>{statusText(day, t)}</span>
      </div>
      {day.impacted_by_outage && (
        <div className="outage-warning outage-warning-compact">
          <strong>{t("outage.warningTitle")}</strong>
          <span>{t("outage.warningText")}</span>
        </div>
      )}

      <div className="selected-day-grid">
        <CalendarMetric
          label={t("stats.checkIn")}
          value={day.check_in_at ? formatOnlyTime(day.check_in_at, locale) : t("stats.no")}
          issue={day.late_minutes > 0 ? `${t("stats.late")} +${day.late_minutes} ${t("time.minuteShort")}` : undefined}
        />
        <CalendarMetric
          label={t("stats.checkOut")}
          value={day.check_out_at ? formatOnlyTime(day.check_out_at, locale) : t("stats.no")}
          issue={day.early_leave_minutes > 0 ? `${t("home.earlyLeave")} -${day.early_leave_minutes} ${t("time.minuteShort")}` : undefined}
        />
        <CalendarMetric
          label={t("stats.worked")}
          value={day.worked_minutes > 0 ? minutesToClock(day.worked_minutes) : "0:00"}
        />
      </div>
      <AttendanceExplanationBox day={day} onSubmitted={onSubmitted} />
    </section>
  );
}

function CalendarMetric({
  label,
  value,
  issue,
}: {
  label: string;
  value: string;
  issue?: string;
}) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
      {issue && <em className="metric-issue">{issue}</em>}
    </div>
  );
}

function calendarCells(month: Date): Array<{ date: Date } | null> {
  const first = startOfMonth(month);
  const last = endOfMonth(month);
  const firstWeekday = first.getDay() || 7;
  const cells: Array<{ date: Date } | null> = Array.from({ length: firstWeekday - 1 }, () => null);

  for (let day = 1; day <= last.getDate(); day += 1) {
    cells.push({ date: new Date(month.getFullYear(), month.getMonth(), day) });
  }

  while (cells.length % 7 !== 0) {
    cells.push(null);
  }

  return cells;
}

function startOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

function endOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0);
}

function addMonths(date: Date, months: number): Date {
  return new Date(date.getFullYear(), date.getMonth() + months, 1);
}

function formatISODate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function emptyDay(date: string): AttendanceDaySummary {
  return {
    date,
    check_in_at: null,
    check_out_at: null,
    worked_minutes: 0,
    late_minutes: 0,
    early_leave_minutes: 0,
    status: "empty",
    impacted_by_outage: false,
    explanations: [],
  };
}

function monthTitle(
  date: Date,
  formatter: (date: Date, options: Intl.DateTimeFormatOptions) => string,
): string {
  return formatter(date, { month: "long" });
}

function formatDayLong(
  value: string,
  formatter: (value: string, options: Intl.DateTimeFormatOptions) => string,
): string {
  return formatter(value, {
    day: "numeric",
    month: "long",
    weekday: "long",
  });
}

function formatOnlyTime(value: string, locale: string): string {
  return new Intl.DateTimeFormat(locale, {
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function minutesToClock(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return `${hours}:${String(rest).padStart(2, "0")}`;
}

function statusText(day: AttendanceDaySummary, t: ReturnType<typeof useI18n>["t"]): string {
  if (day.status === "empty") return t("status.empty");
  if (day.status === "in_progress") return t("status.inProgress");
  return t("status.complete");
}


function dayDotClass(day: AttendanceDaySummary): string {
  if (day.impacted_by_outage) return "calendar-dot-outage";
  if (day.explanations.some((item) => item.status === "pending")) return "calendar-dot-outage";
  if (day.late_minutes > 0 || day.early_leave_minutes > 0) return "calendar-dot-issue";
  if (day.status === "in_progress") return "calendar-dot-progress";
  return "calendar-dot-complete";
}
