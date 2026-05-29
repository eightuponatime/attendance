import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { getAttendanceSummary } from "../api/attendanceApi";
import { errorText } from "../../../shared/api/errors";
import { useI18n } from "../../../shared/i18n/i18n";
import type { AttendanceDaySummary, AttendanceSummary } from "../../../shared/types/api";

export function StatsScreen() {
  const { t } = useI18n();
  const [summary, setSummary] = useState<AttendanceSummary | null>(null);
  const [selectedDate, setSelectedDate] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const range = useMemo(() => currentWeekRange(), []);

  useEffect(() => {
    getAttendanceSummary(formatISODate(range.from), formatISODate(range.to))
      .then((data) => {
        setSummary(data);
        setSelectedDate((current) => {
          if (current && data.days.some((day) => day.date === current)) {
            return current;
          }

          const today = formatISODate(new Date());
          return data.days.some((day) => day.date === today) ? today : data.days[0]?.date ?? null;
        });
        setError(null);
      })
      .catch((err: unknown) => setError(errorText(err)));
  }, [range.from, range.to]);

  const selectedDay = useMemo(() => {
    if (!summary || !selectedDate) {
      return null;
    }

    return summary.days.find((day) => day.date === selectedDate) ?? null;
  }, [selectedDate, summary]);

  return (
    <section className="screen-content">
      <div className="stats-header">
        <div>
          <span>{t("stats.title")}</span>
          <strong>{t("stats.subtitle")}</strong>
        </div>
        <button className="period-button" type="button">
          {t("stats.thisWeek")}
        </button>
      </div>

      {error && <p className="error-banner">{error}</p>}

      <section className="stats-card">
        <div className="chart-title-row">
          <h2>{t("stats.week")}</h2>
          {summary && <span>{t("stats.norm")} {minutesToHours(summary.target_minutes_per_day * 5, t("time.hourShort"))}</span>}
        </div>

        {summary ? (
          <WeekChart
            summary={summary}
            selectedDate={selectedDate}
            onSelectDay={setSelectedDate}
          />
        ) : (
          <p className="muted-text">{t("stats.loading")}</p>
        )}
      </section>

      {selectedDay && <SelectedDayCard day={selectedDay} />}
    </section>
  );
}

function WeekChart({
  summary,
  selectedDate,
  onSelectDay,
}: {
  summary: AttendanceSummary;
  selectedDate: string | null;
  onSelectDay: (date: string) => void;
}) {
  const { t, weekdayLabels } = useI18n();
  const maxMinutes = chartMaxMinutes(summary);
  const yAxis = yAxisTicks(maxMinutes, summary.target_minutes_per_day);
  const scrollRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!selectedDate || !scrollRef.current) {
      return;
    }

    const index = summary.days.findIndex((day) => day.date === selectedDate);
    if (index < 0) {
      return;
    }

    const columnWidth = 70;
    const targetLeft = Math.max(index * columnWidth - 110, 0);
    scrollRef.current.scrollTo({ left: targetLeft, behavior: "smooth" });
  }, [selectedDate, summary.days]);

  return (
    <div className="chart-wrap">
      <div ref={scrollRef} className="chart-scroll" aria-label={t("stats.subtitle")}>
        <div className="chart-shell">
          <div className="y-axis" aria-hidden="true">
            {yAxis.map((tick) => (
              <span
                key={tick}
                className={tick === summary.target_minutes_per_day ? "target-axis-label" : undefined}
                style={{ top: `${100 - (tick / maxMinutes) * 100}%` }}
              >
                {minutesToHoursShort(tick, t("time.hourShort"))}
              </span>
            ))}
          </div>

          <div className="chart-content">
            <div className="bar-values-row">
              {summary.days.map((day) => (
                <span key={day.date}>{day.worked_minutes > 0 ? minutesToClock(day.worked_minutes) : ""}</span>
              ))}
            </div>

            <div
              className="chart-plot"
              style={
                {
                  "--target": `${100 - (summary.target_minutes_per_day / maxMinutes) * 100}%`,
                } as CSSProperties
              }
            >
              <div className="target-line" />

              <div className="bars-row">
                {summary.days.map((day) => (
                  <Bar
                    key={day.date}
                    day={day}
                    maxMinutes={maxMinutes}
                    targetMinutes={summary.target_minutes_per_day}
                    selected={day.date === selectedDate}
                    onSelect={() => onSelectDay(day.date)}
                  />
                ))}
              </div>
            </div>

            <div className="bar-labels-row">
              {summary.days.map((day) => (
                <span key={day.date}>{formatShortDay(day.date, weekdayLabels())}</span>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className="chart-legend">
        <span>
          <i className="legend-blue" /> {t("stats.done")}
        </span>
        <span>
          <i className="legend-orange" /> {t("stats.inProgress")}
        </span>
        <span>
          <i className="legend-red" /> {t("stats.underTarget")}
        </span>
        <span>
          <i className="legend-late" /> {t("stats.late")}
        </span>
      </div>
    </div>
  );
}

function Bar({
  day,
  maxMinutes,
  targetMinutes,
  selected,
  onSelect,
}: {
  day: AttendanceDaySummary;
  maxMinutes: number;
  targetMinutes: number;
  selected: boolean;
  onSelect: () => void;
}) {
  const { formatDateString, t } = useI18n();
  const height = Math.max(6, (day.worked_minutes / maxMinutes) * 100);
  const colorClass = barColor(day, targetMinutes);

  return (
    <button
      className={`bar-track ${selected ? "bar-track-selected" : ""}`}
      type="button"
      aria-label={`${t("stats.showDay")} ${formatDayLong(day.date, formatDateString)}`}
      onClick={onSelect}
    >
      <span className={`bar-fill ${colorClass}`} style={{ height: `${height}%` }} />
      {day.late_minutes > 0 && <span className="late-marker" title={`${t("stats.late")} ${day.late_minutes} ${t("time.minuteShort")}`} />}
    </button>
  );
}

function SelectedDayCard({ day }: { day: AttendanceDaySummary }) {
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
        <Metric
          label={t("stats.checkIn")}
          value={day.check_in_at ? formatOnlyTime(day.check_in_at, locale) : t("stats.no")}
          meta={day.late_minutes > 0 ? `${t("stats.late")} +${day.late_minutes} ${t("time.minuteShort")}` : undefined}
          tone={day.late_minutes > 0 ? "issue" : undefined}
        />
        <Metric
          label={t("stats.checkOut")}
          value={day.check_out_at ? formatOnlyTime(day.check_out_at, locale) : t("stats.no")}
          meta={day.early_leave_minutes > 0 ? `${t("home.earlyLeave")} -${day.early_leave_minutes} ${t("time.minuteShort")}` : undefined}
          tone={day.early_leave_minutes > 0 ? "issue" : undefined}
        />
        <Metric label={t("stats.worked")} value={day.worked_minutes > 0 ? minutesToClock(day.worked_minutes) : "0:00"} />
      </div>
    </section>
  );
}

function Metric({
  label,
  value,
  meta,
  tone,
}: {
  label: string;
  value: string;
  meta?: string;
  tone?: "issue";
}) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
      {meta && <em className={tone === "issue" ? "metric-issue" : undefined}>{meta}</em>}
    </div>
  );
}

function barColor(day: AttendanceDaySummary, targetMinutes: number): string {
  if (day.status === "empty") return "bar-empty";
  if (day.status === "in_progress") return "bar-progress";
  if (day.worked_minutes < targetMinutes) return "bar-short";
  return "bar-complete";
}

function currentWeekRange(): { from: Date; to: Date } {
  const today = new Date();
  const day = today.getDay() || 7;
  const from = new Date(today);
  from.setDate(today.getDate() - day + 1);
  const to = new Date(from);
  to.setDate(from.getDate() + 6);
  return { from, to };
}

function formatISODate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function formatShortDay(value: string, labels: string[]): string {
  const date = new Date(`${value}T00:00:00`);
  const index = (date.getDay() || 7) - 1;
  return labels[index] ?? "";
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

function statusText(day: AttendanceDaySummary, t: ReturnType<typeof useI18n>["t"]): string {
  if (day.status === "empty") {
    return t("status.empty");
  }
  if (day.status === "in_progress") {
    return t("status.inProgress");
  }

  return t("status.complete");
}

function minutesToClock(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return `${hours}:${String(rest).padStart(2, "0")}`;
}

function minutesToHours(minutes: number, hourLabel: string): string {
  return `${Math.round(minutes / 60)} ${hourLabel}`;
}

function minutesToHoursShort(minutes: number, hourLabel: string): string {
  return `${Math.round(minutes / 60)}${hourLabel}`;
}

function chartMaxMinutes(summary: AttendanceSummary): number {
  const rawMax = Math.max(
    summary.target_minutes_per_day,
    ...summary.days.map((day) => day.worked_minutes),
    60,
  );

  return Math.ceil(rawMax / 120) * 120;
}

function yAxisTicks(maxMinutes: number, targetMinutes: number): number[] {
  const middle = Math.round(maxMinutes / 2 / 60) * 60;
  return [maxMinutes, targetMinutes, middle, 0].filter(
    (value, index, values) => values.indexOf(value) === index,
  );
}
