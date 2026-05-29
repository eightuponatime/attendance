import type { AttendanceToday } from "../types/api";
import { BUSINESS_TIME_ZONE } from "../config/businessTime";

export function formatClock(date: Date, locale = "ru-RU", timeZone = BUSINESS_TIME_ZONE): string {
  return new Intl.DateTimeFormat(locale, {
    hour: "2-digit",
    minute: "2-digit",
    timeZone,
  }).format(date);
}

export function formatScreenDate(date: Date, locale = "ru-RU"): string {
  const day = new Intl.DateTimeFormat(locale, { day: "numeric" }).format(date);
  const month = new Intl.DateTimeFormat(locale, { month: "long" }).format(date);
  const weekday = new Intl.DateTimeFormat(locale, { weekday: "long" }).format(date);
  return `${day} ${month}, ${weekday}`;
}

export function buildScreenDate(
  date: Date,
  formatter: (date: Date, options: Intl.DateTimeFormatOptions) => string,
  timeZone = BUSINESS_TIME_ZONE,
): string {
  const day = formatter(date, { day: "numeric", timeZone });
  const month = formatter(date, { month: "long", timeZone });
  const weekday = formatter(date, { weekday: "long", timeZone });
  return `${day} ${month}, ${weekday}`;
}

export function formatEventDate(value: string, locale = "ru-RU", timeZone = BUSINESS_TIME_ZONE): string {
  const date = new Date(value);
  const dayMonth = new Intl.DateTimeFormat(locale, {
    day: "numeric",
    month: "long",
    timeZone,
  }).format(date);
  return `${dayMonth}, ${formatClock(date, locale, timeZone)}`;
}

export function workDuration(
  today: AttendanceToday | null,
  labels = { hour: "ч", minute: "мин" },
): string | undefined {
  if (!today?.check_in || !today.check_out) {
    return undefined;
  }

  const start = new Date(today.check_in.event_at).getTime();
  const end = new Date(today.check_out.event_at).getTime();
  const minutes = Math.max(0, Math.round((end - start) / 60_000));
  const hours = Math.floor(minutes / 60);
  const restMinutes = minutes % 60;

  return `${hours} ${labels.hour} ${String(restMinutes).padStart(2, "0")} ${labels.minute}`;
}
