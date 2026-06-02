import { responseErrorMessage } from "../../../shared/api/errors";
import type { User } from "../../../shared/types/api";

export type LoginPayload = {
  email: string;
  password: string;
};

export type RegisterPayload = {
  email: string;
  password: string;
  last_name: string;
  first_name: string;
  middle_name?: string;
};

export async function login(payload: LoginPayload): Promise<User> {
  const response = await fetch("/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(await responseErrorMessage(response));
  }

  return response.json() as Promise<User>;
}

export async function register(payload: RegisterPayload): Promise<User> {
  const response = await fetch("/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(await responseErrorMessage(response));
  }

  return response.json() as Promise<User>;
}

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
