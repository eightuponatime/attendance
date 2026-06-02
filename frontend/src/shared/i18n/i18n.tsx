import { createContext, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";

export type Language = "ru" | "kk" | "en";

const languageStorageKey = "attendance_language";

const languages: Array<{ code: Language; label: string; flag: string }> = [
  { code: "ru", label: "Русский", flag: "🇷🇺" },
  { code: "kk", label: "Қазақша", flag: "🇰🇿" },
  { code: "en", label: "English", flag: "🇬🇧" },
];

const locales: Record<Language, string> = {
  ru: "ru-RU",
  kk: "kk-KZ",
  en: "en-US",
};

const messages = {
  ru: {
    "app.loading": "Загрузка",
    "app.checkingSession": "Проверяем сессию",
    "admin.authLost": "Доступ к админ-панели отозван или сессия истекла. Войдите снова.",
    "auth.title": "Отметка посещаемости",
    "auth.text": "Войдите в приложение",
    "auth.loginText": "Войдите по email и паролю",
    "auth.registerText": "Заполните ФИО и создайте пароль",
    "auth.googleRegistrationText": "Введите личную информацию для завершения регистрации",
    "auth.googleRegistrationNotice": "Google аккаунт подтвержден",
    "auth.googleConfirmed": "Подтвержденный email",
    "auth.adminText": "Войдите через Google аккаунт администратора",
    "auth.login": "Войти через Google",
    "auth.loginSubmit": "Войти",
    "auth.registerSubmit": "Зарегистрироваться",
    "auth.submitting": "Проверяем",
    "auth.switchToRegister": "Первый вход? Зарегистрироваться",
    "auth.switchToLogin": "Уже есть аккаунт? Войти",
    "auth.registerWithPassword": "Зарегистрироваться с паролем",
    "auth.backToLogin": "Вернуться ко входу",
    "auth.email": "Email",
    "auth.password": "Пароль",
    "auth.passwordRepeat": "Повторите пароль",
    "auth.lastName": "Фамилия",
    "auth.firstName": "Имя",
    "auth.middleName": "Отчество",
    "auth.passwordMismatch": "Пароли не совпадают",
    "auth.formError": "Не удалось выполнить вход",
    "auth.adminLogin": "Войти в админ-панель",
    "confirm.cancel": "Отмена",
    "confirm.close": "Закрыть",
    "confirm.logout.title": "Выйти из аккаунта?",
    "confirm.logout.text": "Сессия посещаемости на этом устройстве будет завершена.",
    "confirm.adminLogout.title": "Выйти из админ-панели?",
    "confirm.adminLogout.text": "Админ-сессия на этом устройстве будет завершена. Обычный вход в приложение не изменится.",
    "confirm.logout.submit": "Выйти",
    "nav.home": "Главная",
    "nav.stats": "Статистика",
    "nav.calendar": "Календарь",
    "nav.profile": "Профиль",
    "home.today": "Сегодня",
    "home.status.done": "Рабочий день завершен",
    "home.status.checkedIn": "Приход зафиксирован",
    "home.status.empty": "Вы ещё не отметились",
    "home.checkIn": "Приход",
    "home.checkOut": "Уход",
    "home.marking": "Отмечаем",
    "home.history.checkIn": "Приход",
    "home.history.checkOut": "Уход",
    "home.notMarked": "Не отмечено",
    "home.late": "Опоздание",
    "home.earlyLeave": "Ранний уход",
    "home.hint": "Ваше рабочее время рассчитывается только после отметки «Уход»",
    "outage.warningTitle": "В этот день был сбой сервера",
    "outage.warningText": "Если отметка не сохранилась, HR сможет восстановить данные после проверки.",
    "stats.title": "Статистика",
    "stats.subtitle": "Отработанное время",
    "stats.thisWeek": "На этой неделе",
    "stats.week": "За неделю",
    "stats.norm": "Норма",
    "stats.loading": "Загрузка статистики",
    "stats.done": "Выполнено",
    "stats.inProgress": "В процессе",
    "stats.underTarget": "Меньше нормы",
    "stats.late": "Опоздание",
    "stats.showDay": "Показать день",
    "stats.checkIn": "Приход",
    "stats.checkOut": "Уход",
    "stats.worked": "Отработано",
    "stats.no": "Нет",
    "status.empty": "Нет отметок",
    "status.inProgress": "В процессе",
    "status.complete": "День завершен",
    "calendar.previousYear": "Предыдущий год",
    "calendar.previousMonth": "Предыдущий месяц",
    "calendar.nextYear": "Следующий год",
    "calendar.nextMonth": "Следующий месяц",
    "calendar.monthSelect": "Выбор месяца",
    "profile.language": "Язык",
    "profile.logout": "Выйти",
    "time.hourShort": "ч",
    "time.minuteShort": "мин",
  },
  kk: {
    "app.loading": "Жүктелуде",
    "app.checkingSession": "Сессия тексерілуде",
    "admin.authLost": "Әкімші панеліне кіру рұқсаты қайтарылды немесе сессия аяқталды. Қайта кіріңіз.",
    "auth.title": "Қатысуды белгілеу",
    "auth.text": "Қолданбаға кіріңіз",
    "auth.loginText": "Email және пароль арқылы кіріңіз",
    "auth.registerText": "Аты-жөніңізді толтырып, пароль жасаңыз",
    "auth.googleRegistrationText": "Тіркеуді аяқтау үшін жеке ақпаратыңызды енгізіңіз",
    "auth.googleRegistrationNotice": "Google аккаунты расталды",
    "auth.googleConfirmed": "Расталған email",
    "auth.adminText": "Әкімші Google аккаунтымен кіріңіз",
    "auth.login": "Google арқылы кіру",
    "auth.loginSubmit": "Кіру",
    "auth.registerSubmit": "Тіркелу",
    "auth.submitting": "Тексерілуде",
    "auth.switchToRegister": "Бірінші кіру? Тіркелу",
    "auth.switchToLogin": "Аккаунт бар ма? Кіру",
    "auth.registerWithPassword": "Парольмен тіркелу",
    "auth.backToLogin": "Кіруге оралу",
    "auth.email": "Email",
    "auth.password": "Пароль",
    "auth.passwordRepeat": "Парольді қайталаңыз",
    "auth.lastName": "Тегі",
    "auth.firstName": "Аты",
    "auth.middleName": "Әкесінің аты",
    "auth.passwordMismatch": "Парольдер сәйкес емес",
    "auth.formError": "Кіру мүмкін болмады",
    "auth.adminLogin": "Әкімші панеліне кіру",
    "confirm.cancel": "Болдырмау",
    "confirm.close": "Жабу",
    "confirm.logout.title": "Аккаунттан шығасыз ба?",
    "confirm.logout.text": "Осы құрылғыдағы қатысу сессиясы аяқталады.",
    "confirm.adminLogout.title": "Әкімші панелінен шығасыз ба?",
    "confirm.adminLogout.text": "Осы құрылғыдағы әкімші сессиясы аяқталады. Қарапайым қолданбадағы сессия өзгермейді.",
    "confirm.logout.submit": "Шығу",
    "nav.home": "Басты",
    "nav.stats": "Статистика",
    "nav.calendar": "Күнтізбе",
    "nav.profile": "Профиль",
    "home.today": "Бүгін",
    "home.status.done": "Жұмыс күні аяқталды",
    "home.status.checkedIn": "Келу белгіленді",
    "home.status.empty": "Сіз әлі белгіленбедіңіз",
    "home.checkIn": "Келу",
    "home.checkOut": "Кету",
    "home.marking": "Белгіленуде",
    "home.history.checkIn": "Келу",
    "home.history.checkOut": "Кету",
    "home.notMarked": "Белгі жоқ",
    "home.late": "Кешігу",
    "home.earlyLeave": "Ерте кету",
    "home.hint": "Жұмыс уақыты тек «Кету» белгісінен кейін есептеледі",
    "outage.warningTitle": "Бұл күні серверде ақау болды",
    "outage.warningText": "Белгі сақталмаса, HR тексергеннен кейін деректерді қалпына келтіре алады.",
    "stats.title": "Статистика",
    "stats.subtitle": "Жұмыс уақыты",
    "stats.thisWeek": "Осы аптада",
    "stats.week": "Апта бойынша",
    "stats.norm": "Норма",
    "stats.loading": "Статистика жүктелуде",
    "stats.done": "Орындалды",
    "stats.inProgress": "Процесте",
    "stats.underTarget": "Нормадан аз",
    "stats.late": "Кешігу",
    "stats.showDay": "Күнді көрсету",
    "stats.checkIn": "Келу",
    "stats.checkOut": "Кету",
    "stats.worked": "Жұмыс уақыты",
    "stats.no": "Жоқ",
    "status.empty": "Белгілер жоқ",
    "status.inProgress": "Процесте",
    "status.complete": "Күн аяқталды",
    "calendar.previousYear": "Алдыңғы жыл",
    "calendar.previousMonth": "Алдыңғы ай",
    "calendar.nextYear": "Келесі жыл",
    "calendar.nextMonth": "Келесі ай",
    "calendar.monthSelect": "Ай таңдау",
    "profile.language": "Тіл",
    "profile.logout": "Шығу",
    "time.hourShort": "сағ",
    "time.minuteShort": "мин",
  },
  en: {
    "app.loading": "Loading",
    "app.checkingSession": "Checking session",
    "admin.authLost": "Admin access was revoked or the session expired. Sign in again.",
    "auth.title": "Attendance check-in",
    "auth.text": "Sign in to the app",
    "auth.loginText": "Sign in with email and password",
    "auth.registerText": "Enter your name and create a password",
    "auth.googleRegistrationText": "Enter your personal information to complete registration",
    "auth.googleRegistrationNotice": "Google account confirmed",
    "auth.googleConfirmed": "Confirmed email",
    "auth.adminText": "Sign in with an admin Google account",
    "auth.login": "Sign in with Google",
    "auth.loginSubmit": "Sign in",
    "auth.registerSubmit": "Create account",
    "auth.submitting": "Checking",
    "auth.switchToRegister": "First sign-in? Create account",
    "auth.switchToLogin": "Already registered? Sign in",
    "auth.registerWithPassword": "Register with password",
    "auth.backToLogin": "Back to sign in",
    "auth.email": "Email",
    "auth.password": "Password",
    "auth.passwordRepeat": "Repeat password",
    "auth.lastName": "Last name",
    "auth.firstName": "First name",
    "auth.middleName": "Middle name",
    "auth.passwordMismatch": "Passwords do not match",
    "auth.formError": "Sign-in failed",
    "auth.adminLogin": "Sign in to admin",
    "confirm.cancel": "Cancel",
    "confirm.close": "Close",
    "confirm.logout.title": "Sign out?",
    "confirm.logout.text": "The attendance session on this device will be ended.",
    "confirm.adminLogout.title": "Sign out of admin?",
    "confirm.adminLogout.text": "The admin session on this device will be ended. Your regular app session will not change.",
    "confirm.logout.submit": "Sign out",
    "nav.home": "Home",
    "nav.stats": "Stats",
    "nav.calendar": "Calendar",
    "nav.profile": "Profile",
    "home.today": "Today",
    "home.status.done": "Workday completed",
    "home.status.checkedIn": "Check-in recorded",
    "home.status.empty": "You have not checked in yet",
    "home.checkIn": "Check in",
    "home.checkOut": "Check out",
    "home.marking": "Saving",
    "home.history.checkIn": "Check in",
    "home.history.checkOut": "Check out",
    "home.notMarked": "Not marked",
    "home.late": "Late",
    "home.earlyLeave": "Early leave",
    "home.hint": "Your worked time is calculated only after check-out",
    "outage.warningTitle": "There was a server outage on this day",
    "outage.warningText": "If a mark was not saved, HR can restore the data after review.",
    "stats.title": "Stats",
    "stats.subtitle": "Worked time",
    "stats.thisWeek": "This week",
    "stats.week": "Week",
    "stats.norm": "Target",
    "stats.loading": "Loading stats",
    "stats.done": "Done",
    "stats.inProgress": "In progress",
    "stats.underTarget": "Under target",
    "stats.late": "Late",
    "stats.showDay": "Show day",
    "stats.checkIn": "Check in",
    "stats.checkOut": "Check out",
    "stats.worked": "Worked",
    "stats.no": "No",
    "status.empty": "No marks",
    "status.inProgress": "In progress",
    "status.complete": "Day completed",
    "calendar.previousYear": "Previous year",
    "calendar.previousMonth": "Previous month",
    "calendar.nextYear": "Next year",
    "calendar.nextMonth": "Next month",
    "calendar.monthSelect": "Choose month",
    "profile.language": "Language",
    "profile.logout": "Sign out",
    "time.hourShort": "h",
    "time.minuteShort": "min",
  },
} satisfies Record<Language, Record<string, string>>;

type I18nContextValue = {
  language: Language;
  locale: string;
  languages: typeof languages;
  setLanguage: (language: Language) => void;
  t: (key: keyof typeof messages.ru) => string;
  formatDate: (date: Date, options: Intl.DateTimeFormatOptions) => string;
  formatDateString: (value: string, options: Intl.DateTimeFormatOptions) => string;
  weekdayLabels: () => string[];
};

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>(() => initialLanguage());

  const value = useMemo<I18nContextValue>(() => ({
    language,
    locale: locales[language],
    languages,
    setLanguage: (nextLanguage) => {
      window.localStorage.setItem(languageStorageKey, nextLanguage);
      setLanguageState(nextLanguage);
    },
    t: (key) => messages[language][key] ?? messages.ru[key],
    formatDate: (date, options) => formatDateForLanguage(language, date, options),
    formatDateString: (value, options) => formatDateForLanguage(language, new Date(`${value}T00:00:00`), options),
    weekdayLabels: () => weekdayLabelsForLanguage(language),
  }), [language]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used inside I18nProvider");
  }

  return context;
}

function initialLanguage(): Language {
  const saved = window.localStorage.getItem(languageStorageKey);
  if (isLanguage(saved)) {
    return saved;
  }

  for (const candidate of window.navigator.languages ?? [window.navigator.language]) {
    const base = candidate.toLowerCase().split("-")[0];
    if (isLanguage(base)) {
      return base;
    }
  }

  return "ru";
}

function isLanguage(value: string | null | undefined): value is Language {
  return value === "ru" || value === "kk" || value === "en";
}

function formatDateForLanguage(
  language: Language,
  date: Date,
  options: Intl.DateTimeFormatOptions,
): string {
  if (language !== "kk") {
    return new Intl.DateTimeFormat(locales[language], options).format(date);
  }

  const partsForZone = datePartsForTimeZone(date, options.timeZone);
  const month = options.month ? kkMonths[partsForZone.monthIndex] : "";
  const weekday = options.weekday ? kkWeekdaysLong[partsForZone.weekdayIndex] : "";
  const day = options.day ? String(partsForZone.day) : "";
  const parts = [day, month, weekday].filter(Boolean);
  return parts.join(options.weekday && (options.day || options.month) ? ", " : " ");
}

function datePartsForTimeZone(
  date: Date,
  timeZone: Intl.DateTimeFormatOptions["timeZone"],
): { day: number; monthIndex: number; weekdayIndex: number } {
  if (!timeZone) {
    return {
      day: date.getDate(),
      monthIndex: date.getMonth(),
      weekdayIndex: date.getDay() || 7,
    };
  }

  const parts = new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "numeric",
    weekday: "short",
    timeZone,
  }).formatToParts(date);
  const day = Number(parts.find((part) => part.type === "day")?.value ?? date.getDate());
  const month = Number(parts.find((part) => part.type === "month")?.value ?? date.getMonth() + 1);
  const weekday = parts.find((part) => part.type === "weekday")?.value ?? "";

  return {
    day,
    monthIndex: month - 1,
    weekdayIndex: weekdayIndexByShortName[weekday] ?? (date.getDay() || 7),
  };
}

function weekdayLabelsForLanguage(language: Language): string[] {
  if (language === "kk") {
    return kkWeekdaysShort;
  }

  const monday = new Date(2026, 0, 5);
  return Array.from({ length: 7 }, (_, index) =>
    new Intl.DateTimeFormat(locales[language], { weekday: "short" })
      .format(new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + index))
      .replace(".", ""),
  );
}

const kkMonths = [
  "қаңтар",
  "ақпан",
  "наурыз",
  "сәуір",
  "мамыр",
  "маусым",
  "шілде",
  "тамыз",
  "қыркүйек",
  "қазан",
  "қараша",
  "желтоқсан",
];

const kkWeekdaysLong: Record<number, string> = {
  1: "дүйсенбі",
  2: "сейсенбі",
  3: "сәрсенбі",
  4: "бейсенбі",
  5: "жұма",
  6: "сенбі",
  7: "жексенбі",
};

const weekdayIndexByShortName: Record<string, number> = {
  Mon: 1,
  Tue: 2,
  Wed: 3,
  Thu: 4,
  Fri: 5,
  Sat: 6,
  Sun: 7,
};

const kkWeekdaysShort = ["дс", "сс", "ср", "бс", "жм", "сб", "жс"];
