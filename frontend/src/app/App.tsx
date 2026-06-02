import { useCallback, useEffect, useMemo, useState } from "react";
import { adminLogout, getAdminMe, isAdminAuthError } from "../features/admin/api/adminApi";
import { AdminDashboard } from "../features/admin/ui/AdminDashboard";
import { checkIn, checkOut, getAttendanceToday } from "../features/attendance/api/attendanceApi";
import { CalendarScreen } from "../features/attendance/ui/CalendarScreen";
import { HomeScreen } from "../features/attendance/ui/HomeScreen";
import { StatsScreen } from "../features/attendance/ui/StatsScreen";
import { getMe, logout } from "../features/auth/api/authApi";
import { LoginScreen } from "../features/auth/ui/LoginScreen";
import { ProfileScreen } from "../features/profile/ui/ProfileScreen";
import { errorText } from "../shared/api/errors";
import { browserName, getDeviceId, phoneModel } from "../shared/device/device";
import { buildScreenDate } from "../shared/lib/date";
import { BUSINESS_TIME_ZONE } from "../shared/config/businessTime";
import { useNow } from "../shared/lib/useNow";
import { I18nProvider, useI18n } from "../shared/i18n/i18n";
import type { AdminMe, AttendanceMarkPayload, AttendanceToday, User } from "../shared/types/api";
import { CenteredState } from "../shared/ui/CenteredState";
import { ConfirmDialog } from "../shared/ui/ConfirmDialog";
import { BottomNav, type Tab } from "./BottomNav";

type LoadState = "loading" | "ready" | "guest";
type AdminState = "unknown" | "checking" | "allowed" | "denied";

export function App() {
  return (
    <I18nProvider>
      <AppContent />
    </I18nProvider>
  );
}

function AppContent() {
  const { formatDate, t } = useI18n();
  const isAdminRoute = window.location.pathname.startsWith("/admin");
  const [authError] = useState(() => consumeAuthErrorFromURL());
  const [authNotice] = useState(() => consumeAuthNoticeFromURL());
  const [deviceId] = useState(() => getDeviceId());
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [adminState, setAdminState] = useState<AdminState>("unknown");
  const [admin, setAdmin] = useState<AdminMe | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [today, setToday] = useState<AttendanceToday | null>(null);
  const [tab, setTab] = useState<Tab>("home");
  const [isMarking, setIsMarking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmAction, setConfirmAction] = useState<null | "user-logout" | "admin-logout">(null);

  const refresh = useCallback(async () => {
    setError(null);
    if (isAdminRoute) {
      setToday(null);
      setUser(null);
      setAdminState("checking");
      try {
        const adminMe = await getAdminMe();
        setAdmin(adminMe);
        setAdminState(adminMe.is_admin ? "allowed" : "denied");
        setLoadState("ready");
      } catch {
        setAdmin(null);
        setAdminState("denied");
        setError(authError);
        setLoadState("guest");
      }
      return;
    }

    const me = await getMe();
    setUser(me);

    if (!me) {
      setToday(null);
      setError(authError);
      setLoadState("guest");
      return;
    }

    const attendance = await getAttendanceToday();
    setToday(attendance);

    setLoadState("ready");
  }, [authError, isAdminRoute]);

  useEffect(() => {
    refresh().catch((err: unknown) => {
      setError(errorText(err));
      setLoadState("guest");
    });
  }, [refresh]);

  useEffect(() => {
    const syncOnVisible = () => {
      if (isAdminRoute) {
        return;
      }
      if (document.visibilityState === "visible") {
        refresh().catch((err: unknown) => setError(errorText(err)));
      }
    };

    document.addEventListener("visibilitychange", syncOnVisible);
    window.addEventListener("focus", syncOnVisible);

    return () => {
      document.removeEventListener("visibilitychange", syncOnVisible);
      window.removeEventListener("focus", syncOnVisible);
    };
  }, [isAdminRoute, refresh]);

  const now = useNow();
  const screenDate = useMemo(() => buildScreenDate(now, formatDate, BUSINESS_TIME_ZONE), [formatDate, now]);

  const mark = async (type: "check_in" | "check_out") => {
    setError(null);
    setIsMarking(true);

    try {
      const payload: AttendanceMarkPayload = {
        device_id: deviceId,
        phone_model: phoneModel(),
        browser: browserName(),
      };

      const nextToday = type === "check_in" ? await checkIn(payload) : await checkOut(payload);
      setToday(nextToday);
    } catch (err: unknown) {
      setError(errorText(err));
    } finally {
      setIsMarking(false);
    }
  };

  const signOut = async () => {
    setError(null);
    try {
      await logout();
      setUser(null);
      setToday(null);
      setLoadState("guest");
      setTab("home");
    } catch (err: unknown) {
      setError(errorText(err));
    }
  };

  const signOutAdmin = async () => {
    setError(null);
    try {
      await adminLogout();
    } catch (err: unknown) {
      if (isAdminAuthError(err)) {
        handleAdminAuthLost();
        return;
      }
      setError(errorText(err));
      return;
    }

    setAdmin(null);
    setAdminState("denied");
    setLoadState("guest");
  };

  const handleAdminAuthLost = () => {
    setAdmin(null);
    setAdminState("denied");
    setLoadState("guest");
    setError(t("admin.authLost"));
  };

  const confirmDialog =
    confirmAction === "admin-logout"
      ? {
          title: t("confirm.adminLogout.title"),
          text: t("confirm.adminLogout.text"),
          confirmText: t("confirm.logout.submit"),
          onConfirm: () => {
            setConfirmAction(null);
            void signOutAdmin();
          },
        }
      : {
          title: t("confirm.logout.title"),
          text: t("confirm.logout.text"),
          confirmText: t("confirm.logout.submit"),
          onConfirm: () => {
            setConfirmAction(null);
            void signOut();
          },
        };

  return (
    <main className={isAdminRoute ? "admin-shell" : "app-shell"}>
      <ConfirmDialog
        open={confirmAction !== null}
        title={confirmDialog.title}
        text={confirmDialog.text}
        confirmText={confirmDialog.confirmText}
        tone="danger"
        onConfirm={confirmDialog.onConfirm}
        onCancel={() => setConfirmAction(null)}
      />

      {loadState === "loading" ? (
        <CenteredState title={t("app.loading")} text={t("app.checkingSession")} />
      ) : loadState === "guest" ? (
        <LoginScreen error={error} notice={authNotice} admin={isAdminRoute} onAuthenticated={() => void refresh()} />
      ) : isAdminRoute ? (
        adminState === "checking" || adminState === "unknown" ? (
          <CenteredState title="Загрузка" text="Проверяем доступ к админ-панели" />
        ) : adminState === "denied" ? (
          <CenteredState title="Доступ закрыт" text="Эта панель доступна только администраторам" />
        ) : (
          <AdminDashboard
            user={admin}
            onLogout={() => setConfirmAction("admin-logout")}
            onAuthLost={handleAdminAuthLost}
          />
        )
      ) : (
        <>
          {tab === "home" && (
            <HomeScreen
              dateLabel={screenDate}
              now={now}
              today={today}
              error={error}
              isMarking={isMarking}
              onCheckIn={() => void mark("check_in")}
              onCheckOut={() => void mark("check_out")}
            />
          )}

          {tab === "stats" && <StatsScreen />}

          {tab === "calendar" && <CalendarScreen />}

          {tab === "profile" && user && (
            <ProfileScreen user={user} onLogout={() => setConfirmAction("user-logout")} />
          )}

          <BottomNav active={tab} onChange={setTab} />
        </>
      )}
    </main>
  );
}

function consumeAuthErrorFromURL(): string | null {
  const url = new URL(window.location.href);
  const message = url.searchParams.get("auth_error");
  if (!message) {
    return null;
  }

  url.searchParams.delete("auth_error");
  window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`);
  return message;
}

function consumeAuthNoticeFromURL(): string | null {
  const url = new URL(window.location.href);
  const message = url.searchParams.get("auth_notice");
  if (!message) {
    return null;
  }

  url.searchParams.delete("auth_notice");
  window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`);
  return message;
}
