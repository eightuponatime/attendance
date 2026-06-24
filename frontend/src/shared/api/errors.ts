type Translate = (key: any) => string;

export async function responseErrorMessage(response: Response): Promise<string> {
  try {
    const data = (await response.json()) as { error?: string };
    return data.error ?? "Запрос не выполнен";
  } catch {
    return "Запрос не выполнен";
  }
}

export function errorText(err: unknown, t?: Translate): string {
  if (err instanceof GeolocationPositionError) {
    return translateErrorMessage("Разрешите доступ к геолокации", t) ?? "Разрешите доступ к геолокации";
  }
  if (err instanceof Error) {
    return translateErrorMessage(err.message, t) ?? err.message;
  }

  return translateErrorMessage("Что-то пошло не так", t) ?? "Что-то пошло не так";
}

export function translateErrorMessage(message: string | null | undefined, t?: Translate): string | null {
  if (!message) return null;
  const key = errorMessageKey(message);
  return key && t ? t(key) : message;
}

function errorMessageKey(message: string): string | null {
  const normalized = message.trim();
  const exact: Record<string, string> = {
    "Запрос не выполнен": "error.requestFailed",
    "Что-то пошло не так": "error.generic",
    "Разрешите доступ к геолокации": "error.geolocationPermission",
    "Не удалось получить пользователя": "error.userLoadFailed",
    "Не удалось получить статус дня": "error.attendanceTodayFailed",
    "Не удалось получить статистику": "error.attendanceSummaryFailed",
    "Аккаунт с таким email уже существует": "error.emailAlreadyExists",
    "Аккаунт с таким email не найден": "error.accountNotFound",
    "Этот email зарегистрирован через Google. Нажмите «Войти через Google» ниже.": "error.googleOnlyAccount",
    "Неверный пароль": "error.passwordMismatch",
    "Проверьте email и пароль": "error.invalidCredentials",
    "Не удалось выполнить вход": "error.authFailed",
    "Не удалось подтвердить вход через Google": "error.googleConfirmFailed",
    "Не удалось получить профиль Google": "error.googleProfileFailed",
    "Не удалось войти через Google": "error.googleLoginFailed",
    "Не удалось создать сессию": "error.sessionCreateFailed",
    "check-in is required before check-out": "error.checkInRequired",
    "attendance event already exists": "error.attendanceAlreadyExists",
    "explanation is not available for this day": "error.explanationUnavailable",
    "invalid request body": "error.invalidRequest",
    "unauthorized": "error.unauthorized",
    "auth failed": "error.authFailed",
  };

  if (exact[normalized]) {
    return exact[normalized];
  }

  if (normalized.includes("email is invalid")) return "error.emailInvalid";
  if (normalized.includes("password is too short")) return "error.passwordTooShort";
  if (normalized.includes("last_name is empty")) return "error.lastNameRequired";
  if (normalized.includes("first_name is empty")) return "error.firstNameRequired";
  if (normalized.includes("comment is empty")) return "error.commentRequired";
  if (normalized.includes("comment is too long")) return "error.commentTooLong";

  return null;
}
