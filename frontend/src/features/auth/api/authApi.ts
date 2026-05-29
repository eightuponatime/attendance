import { responseErrorMessage } from "../../../shared/api/errors";
import type { User } from "../../../shared/types/api";

export async function getMe(): Promise<User | null> {
  const response = await fetch("/api/me", {
    credentials: "include",
  });

  if (response.status === 401) {
    return null;
  }
  if (!response.ok) {
    throw new Error("Не удалось получить пользователя");
  }

  return response.json() as Promise<User>;
}

export async function logout(): Promise<void> {
  const response = await fetch("/api/logout", {
    method: "POST",
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error(await responseErrorMessage(response));
  }
}
