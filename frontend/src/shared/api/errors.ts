export async function responseErrorMessage(response: Response): Promise<string> {
  try {
    const data = (await response.json()) as { error?: string };
    return data.error ?? "Запрос не выполнен";
  } catch {
    return "Запрос не выполнен";
  }
}

export function errorText(err: unknown): string {
  if (err instanceof GeolocationPositionError) {
    return "Разрешите доступ к геолокации";
  }
  if (err instanceof Error) {
    return err.message;
  }

  return "Что-то пошло не так";
}
