import { useEffect, useState } from "react";
import { submitAttendanceExplanations } from "../api/attendanceApi";
import { errorText } from "../../../shared/api/errors";
import { useI18n } from "../../../shared/i18n/i18n";
import type {
  AttendanceDaySummary,
  AttendanceExplanation,
  AttendanceExplanationReason,
} from "../../../shared/types/api";

export function AttendanceExplanationBox({
  day,
  days = [day],
  onSubmitted,
}: {
  day: AttendanceDaySummary;
  days?: AttendanceDaySummary[];
  onSubmitted: () => void;
}) {
  const { t } = useI18n();
  const existingByReason = new Map(day.explanations.map((item) => [item.reason_type, item]));
  const availableReasons = explanationReasons(day, t).filter((reason) => !existingByReason.has(reason.value));
  const firstReason = availableReasons[0]?.value ?? "late";
  const outageMissingDates = outageMissingExplanationDates(days);
  const [reasonType, setReasonType] = useState<AttendanceExplanationReason>(firstReason);
  const [selectedDates, setSelectedDates] = useState<string[]>([day.date]);
  const [comment, setComment] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const isGroupedOutageReason = reasonType === "missing_day" && isEmptyOutageDay(day) && outageMissingDates.length > 1;
  const isVoidDayReason = reasonType === "void_day_request";
  const submitDates = isGroupedOutageReason ? selectedDates : [day.date];

  useEffect(() => {
    const nextReason = availableReasons[0]?.value ?? "late";
    setReasonType(nextReason);
    setSelectedDates([day.date]);
    setComment("");
    setError(null);
  }, [day.date, day.explanations.length]);

  const submit = async () => {
    if (!comment.trim() || availableReasons.length === 0 || submitDates.length === 0) return;
    setSaving(true);
    try {
      await submitAttendanceExplanations(submitDates.map((date) => ({
        business_date: date,
        reason_type: reasonType,
        comment,
      })));
      setComment("");
      setError(null);
      onSubmitted();
    } catch (err: unknown) {
      setError(errorText(err));
    } finally {
      setSaving(false);
    }
  };

  const toggleDate = (date: string) => {
    setSelectedDates((current) => {
      if (current.includes(date) && current.length > 1) {
        return current.filter((item) => item !== date);
      }
      if (current.includes(date)) return current;
      return [...current, date].sort();
    });
  };

  if (availableReasons.length === 0 && day.explanations.length === 0) {
    return null;
  }

  return (
    <div className="explanation-box">
      <div className="explanation-header">
        <div>
          <span>{t("explanation.title")}</span>
          <strong>{availableReasons.length > 0 ? t("explanation.available") : t("explanation.existing")}</strong>
        </div>
      </div>

      {day.explanations.length > 0 && (
        <div className="explanation-history">
          {day.explanations.map((item) => (
            <div key={item.id} className="explanation-history-row">
              <div>
                <strong>{reasonText(item.reason_type, t)}</strong>
                <span>{item.comment}</span>
                {item.review_note && (
                  <div className="explanation-admin-reply">
                    <small>{t("explanation.hrReply")}</small>
                    <p>{item.review_note}</p>
                  </div>
                )}
              </div>
              <ExplanationStatus explanation={item} t={t} />
            </div>
          ))}
        </div>
      )}

      {availableReasons.length > 0 && (
        <>
          <div className="explanation-reason-tabs">
            {availableReasons.map((reason) => (
              <button
                key={reason.value}
                className={[
                  reason.value === reasonType ? "explanation-reason-active" : "",
                  reason.value === "void_day_request" ? "explanation-reason-danger" : "",
                ].filter(Boolean).join(" ")}
                type="button"
                onClick={() => setReasonType(reason.value)}
              >
                {reason.label}
              </button>
            ))}
          </div>
          {isVoidDayReason && (
            <div className="explanation-danger-note">
              <strong>День будет исключен из статистики</strong>
              <span>Используйте это, если день попал в учет по ошибке и не должен влиять на часы, опоздания и ранние уходы.</span>
            </div>
          )}
          {isGroupedOutageReason && (
            <div className="explanation-date-select">
              <span>{t("explanation.outageDates")}</span>
              <div>
                {outageMissingDates.map((date) => (
                  <label key={date}>
                    <input
                      type="checkbox"
                      checked={selectedDates.includes(date)}
                      onChange={() => toggleDate(date)}
                    />
                    {formatDisplayDate(date)}
                  </label>
                ))}
              </div>
            </div>
          )}
          <textarea
            value={comment}
            rows={3}
            maxLength={1000}
            placeholder={t("explanation.placeholder")}
            onChange={(event) => setComment(event.target.value)}
          />
          {error && <p className="form-error">{error}</p>}
          <button type="button" disabled={saving || !comment.trim() || submitDates.length === 0} onClick={() => void submit()}>
            {saving ? t("explanation.sending") : t("explanation.submit")}
          </button>
        </>
      )}
    </div>
  );
}

function outageMissingExplanationDates(days: AttendanceDaySummary[]): string[] {
  return days
    .filter((item) => (
      isEmptyOutageDay(item) &&
      canExplainClosedDay(item.date) &&
      !isWeekend(item.date) &&
      !item.explanations.some((explanation) => explanation.reason_type === "missing_day")
    ))
    .map((item) => item.date)
    .sort();
}

function isEmptyOutageDay(day: AttendanceDaySummary): boolean {
  return day.impacted_by_outage && !day.check_in_at && !day.check_out_at;
}

function ExplanationStatus({
  explanation,
  t,
}: {
  explanation: AttendanceExplanation;
  t: ReturnType<typeof useI18n>["t"];
}) {
  return (
    <span className={`explanation-status explanation-status-${explanation.status}`}>
      {explanationStatusText(explanation.status, t)}
    </span>
  );
}

function explanationReasons(
  day: AttendanceDaySummary,
  t: ReturnType<typeof useI18n>["t"],
): Array<{ value: AttendanceExplanationReason; label: string }> {
  const reasons: Array<{ value: AttendanceExplanationReason; label: string }> = [];
  const canExplainMissing = canExplainClosedDay(day.date);
  const emptyOutageDay = isEmptyOutageDay(day);

  if (day.late_minutes > 0 && day.check_in_at) {
    reasons.push({ value: "late", label: reasonText("late", t) });
  }
  if (day.early_leave_minutes > 0 && day.check_out_at) {
    reasons.push({ value: "early_leave", label: reasonText("early_leave", t) });
  }
  if (canExplainMissing && !day.check_in_at && day.check_out_at) {
    reasons.push({ value: "missing_check_in", label: reasonText("missing_check_in", t) });
  }
  if (canExplainMissing && day.check_in_at && !day.check_out_at) {
    reasons.push({ value: "missing_check_out", label: reasonText("missing_check_out", t) });
  }
  if (canExplainMissing && !isWeekend(day.date) && !day.check_in_at && !day.check_out_at) {
    reasons.push({
      value: "missing_day",
      label: emptyOutageDay ? t("explanation.reason.outage_missing_day") : reasonText("missing_day", t),
    });
  }
  if (canExplainMissing && !emptyOutageDay && canRequestVoidDay(day)) {
    reasons.push({ value: "void_day_request", label: reasonText("void_day_request", t) });
  }

  return reasons;
}

function canRequestVoidDay(day: AttendanceDaySummary): boolean {
  if (!isWeekend(day.date)) return true;
  return Boolean(day.check_in_at || day.check_out_at);
}

function canExplainClosedDay(date: string): boolean {
  const today = formatISODate(new Date());
  if (date < today) return true;
  if (date > today) return false;
  return new Date().getHours() >= 18;
}

function isWeekend(value: string): boolean {
  const weekday = new Date(`${value}T00:00:00`).getDay();
  return weekday === 0 || weekday === 6;
}

function reasonText(value: AttendanceExplanationReason, t: ReturnType<typeof useI18n>["t"]): string {
  switch (value) {
    case "late":
      return t("explanation.reason.late");
    case "early_leave":
      return t("explanation.reason.early_leave");
    case "missing_check_in":
      return t("explanation.reason.missing_check_in");
    case "missing_check_out":
      return t("explanation.reason.missing_check_out");
    case "missing_day":
      return t("explanation.reason.missing_day");
    case "void_day_request":
      return t("explanation.reason.void_day_request");
  }
}

function explanationStatusText(
  value: AttendanceExplanation["status"],
  t: ReturnType<typeof useI18n>["t"],
): string {
  switch (value) {
    case "pending":
      return t("explanation.status.pending");
    case "approved":
      return t("explanation.status.approved");
    case "rejected":
      return t("explanation.status.rejected");
  }
}

function formatISODate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function formatDisplayDate(value: string): string {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "short",
  }).format(new Date(`${value}T00:00:00`));
}
