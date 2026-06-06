import { responseErrorMessage } from "../../../shared/api/errors";
import type {
  AttendanceExplanation,
  AttendanceExplanationReason,
  AttendanceMarkPayload,
  AttendanceSummary,
  AttendanceToday,
} from "../../../shared/types/api";

const jsonHeaders = {
  "Content-Type": "application/json",
};

export async function getAttendanceToday(): Promise<AttendanceToday> {
  const response = await fetch("/api/attendance/today", {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Не удалось получить статус дня");
  }

  return response.json() as Promise<AttendanceToday>;
}

export async function getAttendanceSummary(from: string, to: string): Promise<AttendanceSummary> {
  const params = new URLSearchParams({ from, to });
  const response = await fetch(`/api/attendance/summary?${params.toString()}`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Не удалось получить статистику");
  }

  return response.json() as Promise<AttendanceSummary>;
}

export async function checkIn(payload: AttendanceMarkPayload): Promise<AttendanceToday> {
  return markAttendance("/api/attendance/check-in", payload);
}

export async function checkOut(payload: AttendanceMarkPayload): Promise<AttendanceToday> {
  return markAttendance("/api/attendance/check-out", payload);
}

export async function submitAttendanceExplanation(payload: {
  business_date: string;
  reason_type: AttendanceExplanationReason;
  comment: string;
}): Promise<AttendanceExplanation> {
  const response = await fetch("/api/attendance/explanations", {
    method: "POST",
    headers: jsonHeaders,
    credentials: "include",
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(await responseErrorMessage(response));
  }

  return response.json() as Promise<AttendanceExplanation>;
}

async function markAttendance(
  url: string,
  payload: AttendanceMarkPayload,
): Promise<AttendanceToday> {
  const response = await fetch(url, {
    method: "POST",
    headers: jsonHeaders,
    credentials: "include",
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(await responseErrorMessage(response));
  }

  return response.json() as Promise<AttendanceToday>;
}
