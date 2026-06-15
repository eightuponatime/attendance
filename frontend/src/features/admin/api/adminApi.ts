import type {
  AdminAccess,
  AdminAccessList,
  AdminEmployeeMonthDetail,
  AdminEmployeesMonth,
  AdminExplanationDecision,
  AdminExplanationList,
  AdminMe,
  AdminReportList,
  AdminSessionList,
  AdminSuspiciousActivity,
  AdminOutageDay,
  AdminOutageRepairItem,
  AdminSystemOutageList,
} from "../../../shared/types/api";
import { responseErrorMessage } from "../../../shared/api/errors";

export class AdminAuthError extends Error {
  constructor(message = "Доступ к админ-панели больше не активен") {
    super(message);
    this.name = "AdminAuthError";
  }
}

export function isAdminAuthError(err: unknown): err is AdminAuthError {
  return err instanceof AdminAuthError;
}

async function assertAdminResponse(response: Response, fallback: string): Promise<void> {
  if (response.ok) {
    return;
  }

  if (response.status === 401 || response.status === 403) {
    throw new AdminAuthError();
  }

  throw new Error((await responseErrorMessage(response)) || fallback);
}

export async function getAdminMe(): Promise<AdminMe> {
  const response = await fetch("/api/admin/me", {
    credentials: "include",
  });

  await assertAdminResponse(response, "Нет доступа к админ-панели");

  return response.json() as Promise<AdminMe>;
}

export async function adminLogout(): Promise<void> {
  const response = await fetch("/api/admin/logout", {
    method: "POST",
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось выйти из админ-панели");
}

export async function getAdminReports(): Promise<AdminReportList> {
  const response = await fetch("/api/admin/reports", {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось получить архив отчетов");

  return response.json() as Promise<AdminReportList>;
}

export async function downloadAdminExcelReport(month: string): Promise<void> {
  const response = await fetch(`/api/admin/reports/excel?${new URLSearchParams({ month })}`, {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось скачать Excel отчет");

  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filenameFromDisposition(response.headers.get("Content-Disposition")) ?? `attendance-${month}.xls`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}

function filenameFromDisposition(value: string | null): string | null {
  if (!value) return null;
  const match = /filename="?([^";]+)"?/i.exec(value);
  return match?.[1] ?? null;
}

export async function getAdminAccess(): Promise<AdminAccessList> {
  const response = await fetch("/api/admin/access", {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось получить список администраторов");

  return response.json() as Promise<AdminAccessList>;
}

export async function addAdminAccess(email: string): Promise<AdminAccess> {
  const response = await fetch("/api/admin/access", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ email }),
  });

  await assertAdminResponse(response, "Не удалось добавить администратора");

  return response.json() as Promise<AdminAccess>;
}

export async function revokeAdminAccess(email: string): Promise<void> {
  const response = await fetch(`/api/admin/access/${encodeURIComponent(email)}`, {
    method: "DELETE",
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось удалить администратора");
}

export async function getAdminSessions(): Promise<AdminSessionList> {
  const response = await fetch("/api/admin/sessions", {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось получить список сессий");

  return response.json() as Promise<AdminSessionList>;
}

export async function revokeAdminSession(sessionId: string): Promise<void> {
  const response = await fetch(`/api/admin/sessions/${sessionId}/revoke`, {
    method: "POST",
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось отозвать сессию");
}

export async function getAdminEmployees(month: string): Promise<AdminEmployeesMonth> {
  const response = await fetch(`/api/admin/employees?${new URLSearchParams({ month })}`, {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось получить список сотрудников");

  return response.json() as Promise<AdminEmployeesMonth>;
}

export async function getAdminEmployee(
  userId: string,
  month: string,
): Promise<AdminEmployeeMonthDetail> {
  const response = await fetch(`/api/admin/employees/${userId}?${new URLSearchParams({ month })}`, {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось получить карточку сотрудника");

  return response.json() as Promise<AdminEmployeeMonthDetail>;
}

export async function getAdminSuspiciousActivity(
  month: string,
): Promise<AdminSuspiciousActivity> {
  const response = await fetch(`/api/admin/suspicious-activity?${new URLSearchParams({ month })}`, {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось получить подозрительную активность");

  return response.json() as Promise<AdminSuspiciousActivity>;
}

export async function getAdminSystemOutages(month: string): Promise<AdminSystemOutageList> {
  const response = await fetch(`/api/admin/system-outages?${new URLSearchParams({ month })}`, {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось получить список сбоев сервера");

  return response.json() as Promise<AdminSystemOutageList>;
}

export async function getAdminExplanations(month: string, status = ""): Promise<AdminExplanationList> {
  const params = new URLSearchParams({ month });
  if (status) params.set("status", status);
  const response = await fetch(`/api/admin/explanations?${params.toString()}`, {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось получить заявки на пересмотр");

  return response.json() as Promise<AdminExplanationList>;
}

export async function approveAdminExplanation(
  explanationId: string,
  payload: AdminExplanationDecision,
): Promise<void> {
  const response = await fetch(`/api/admin/explanations/${explanationId}/approve`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(payload),
  });

  await assertAdminResponse(response, "Не удалось одобрить заявку");
}

export async function rejectAdminExplanation(
  explanationId: string,
  payload: AdminExplanationDecision,
): Promise<void> {
  const response = await fetch(`/api/admin/explanations/${explanationId}/reject`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(payload),
  });

  await assertAdminResponse(response, "Не удалось отклонить заявку");
}

export async function getAdminOutageDay(outageId: string): Promise<AdminOutageDay> {
  const response = await fetch(`/api/admin/system-outages/${outageId}/day`, {
    credentials: "include",
  });

  await assertAdminResponse(response, "Не удалось получить день сбоя");

  return response.json() as Promise<AdminOutageDay>;
}

export async function repairAdminOutage(
  outageId: string,
  resolutionNote: string,
  items: AdminOutageRepairItem[],
): Promise<void> {
  const response = await fetch(`/api/admin/system-outages/${outageId}/repair`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ resolution_note: resolutionNote, items }),
  });

  await assertAdminResponse(response, "Не удалось сохранить исправления");
}
