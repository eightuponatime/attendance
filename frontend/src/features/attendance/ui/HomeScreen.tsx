import { Info, LogIn, LogOut } from "lucide-react";
import {
  formatClock,
  formatEventDate,
  workDuration,
} from "../../../shared/lib/date";
import { useI18n } from "../../../shared/i18n/i18n";
import { BUSINESS_TIME_ZONE } from "../../../shared/config/businessTime";
import { LanguageSwitcher } from "../../../shared/ui/LanguageSwitcher";
import type { AttendanceToday } from "../../../shared/types/api";

type HomeScreenProps = {
  dateLabel: string;
  now: Date;
  today: AttendanceToday | null;
  error: string | null;
  isMarking: boolean;
  onCheckIn: () => void;
  onCheckOut: () => void;
};

export function HomeScreen({
  dateLabel,
  now,
  today,
  error,
  isMarking,
  onCheckIn,
  onCheckOut,
}: HomeScreenProps) {
  const { locale, t } = useI18n();
  const lateMinutes = today ? today.late_minutes || lateMinutesFromCheckIn(today.check_in?.event_at ?? null) : 0;
  const earlyLeaveMinutes = today
    ? today.early_leave_minutes || earlyLeaveMinutesFromCheckOut(today.check_out?.event_at ?? null)
    : 0;
  const statusText = today?.check_in
    ? today.check_out
      ? t("home.status.done")
      : t("home.status.checkedIn")
    : t("home.status.empty");

  return (
    <section className="screen-content">
      <div className="date-row">
        <div className="date-block">
          <span>{t("home.today")}</span>
          <strong>{dateLabel}</strong>
        </div>
        <LanguageSwitcher compact />
      </div>

      <section className="mark-card">
        <p>{statusText}</p>
        <time>{formatClock(now, locale, BUSINESS_TIME_ZONE)}</time>

        <button
          className="mark-button mark-button-in"
          type="button"
          disabled={isMarking || !today?.can_check_in}
          onClick={onCheckIn}
        >
          <span className="mark-icon">
            <LogIn size={28} strokeWidth={3} />
          </span>
          <span>{isMarking && today?.can_check_in ? t("home.marking") : t("home.checkIn")}</span>
        </button>

        <button
          className="mark-button mark-button-out"
          type="button"
          disabled={isMarking || !today?.can_check_out}
          onClick={onCheckOut}
        >
          <span className="mark-icon">
            <LogOut size={28} strokeWidth={3} />
          </span>
          <span>{isMarking && today?.can_check_out ? t("home.marking") : t("home.checkOut")}</span>
        </button>
      </section>

      {error && <p className="error-banner">{error}</p>}

      {today?.impacted_by_outage && (
        <section className="outage-warning" aria-live="polite">
          <strong>{t("outage.warningTitle")}</strong>
          <span>{t("outage.warningText")}</span>
        </section>
      )}

      <section className="today-section">
        <h2>{t("home.today")}</h2>
        <div className="history-card">
          <HistoryRow
            kind="in"
            title={t("home.history.checkIn")}
            eventAt={today?.check_in?.event_at ?? null}
            active={Boolean(today?.check_in)}
            rightText={lateMinutes > 0 ? `+${lateMinutes} ${t("time.minuteShort")}` : undefined}
            rightTone={lateMinutes > 0 ? "late" : undefined}
          />
          <div className="divider" />
          <HistoryRow
            kind="out"
            title={t("home.history.checkOut")}
            eventAt={today?.check_out?.event_at ?? null}
            active={Boolean(today?.check_out)}
            rightText={workDuration(today, { hour: t("time.hourShort"), minute: t("time.minuteShort") })}
            note={earlyLeaveMinutes > 0 ? `${t("home.earlyLeave")} -${earlyLeaveMinutes} ${t("time.minuteShort")}` : undefined}
            noteTone={earlyLeaveMinutes > 0 ? "early" : undefined}
          />
        </div>
      </section>

      <p className="hint">
        <Info size={19} />
        <span>{t("home.hint")}</span>
      </p>
    </section>
  );
}

type HistoryRowProps = {
  kind: "in" | "out";
  title: string;
  eventAt: string | null;
  active: boolean;
  rightText?: string;
  rightTone?: "late";
  note?: string;
  noteTone?: "early";
};

function HistoryRow({ kind, title, eventAt, active, rightText, rightTone, note, noteTone }: HistoryRowProps) {
  const { locale, t } = useI18n();
  return (
    <div className="history-row">
      <div className={`history-icon ${active ? "history-icon-active" : ""}`}>
        {kind === "in" ? (
          <LogIn size={25} strokeWidth={3} />
        ) : (
          <LogOut size={25} strokeWidth={3} />
        )}
      </div>
      <div className="history-main">
        <strong>{title}</strong>
        <span>{eventAt ? formatEventDate(eventAt, locale, BUSINESS_TIME_ZONE) : t("home.notMarked")}</span>
        {note && <em className={noteTone === "early" ? "history-note-early" : undefined}>{note}</em>}
      </div>
      {rightText && (
        <span className={`history-right ${rightTone === "late" ? "history-right-late" : ""}`}>
          {rightText}
        </span>
      )}
    </div>
  );
}

function lateMinutesFromCheckIn(eventAt: string | null): number {
  if (!eventAt) {
    return 0;
  }

  const checkIn = new Date(eventAt);
  const workdayStart = new Date(checkIn);
  workdayStart.setHours(8, 0, 0, 0);

  if (checkIn <= workdayStart) {
    return 0;
  }

  return Math.ceil((checkIn.getTime() - workdayStart.getTime()) / 60_000);
}

function earlyLeaveMinutesFromCheckOut(eventAt: string | null): number {
  if (!eventAt) {
    return 0;
  }

  const checkOut = new Date(eventAt);
  const workdayEnd = new Date(checkOut);
  workdayEnd.setHours(17, 0, 0, 0);

  if (checkOut >= workdayEnd) {
    return 0;
  }

  return Math.ceil((workdayEnd.getTime() - checkOut.getTime()) / 60_000);
}
