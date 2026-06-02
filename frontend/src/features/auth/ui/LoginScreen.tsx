import { useState } from "react";
import type { FormEvent } from "react";
import { Clock3 } from "lucide-react";
import { login, register } from "../api/authApi";
import { useI18n } from "../../../shared/i18n/i18n";
import { LanguageSwitcher } from "../../../shared/ui/LanguageSwitcher";

export function LoginScreen({
  error,
  notice,
  admin = false,
  onAuthenticated,
}: {
  error: string | null;
  notice?: string | null;
  admin?: boolean;
  onAuthenticated?: () => void;
}) {
  const { t } = useI18n();
  const initialAuthParams = new URL(window.location.href).searchParams;
  const [mode, setMode] = useState<"login" | "register">(
    initialAuthParams.get("auth_mode") === "register" ? "register" : "login",
  );
  const [email, setEmail] = useState(initialAuthParams.get("email") ?? "");
  const [password, setPassword] = useState("");
  const [passwordRepeat, setPasswordRepeat] = useState("");
  const [lastName, setLastName] = useState("");
  const [firstName, setFirstName] = useState("");
  const [middleName, setMiddleName] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const googleRegistration = mode === "register" && initialAuthParams.get("auth_mode") === "register" && email !== "";
  const noticeText = notice === "google_registration" ? t("auth.googleRegistrationNotice") : notice;
  const loginPath = admin ? "/auth/admin/google/login" : "/auth/google/login";
  const loginURL = `${loginPath}?${new URLSearchParams({
    return_to: `${window.location.pathname}${window.location.search}`,
  })}`;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);

    if (mode === "register" && !googleRegistration && password !== passwordRepeat) {
      setFormError(t("auth.passwordMismatch"));
      return;
    }

    setSubmitting(true);
    try {
      if (mode === "login") {
        await login({ email, password });
      } else {
        await register({
          email,
          password: googleRegistration ? "" : password,
          last_name: lastName,
          first_name: firstName,
          middle_name: middleName,
        });
      }
      onAuthenticated?.();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : t("auth.formError"));
    } finally {
      setSubmitting(false);
    }
  }

  if (admin) {
    return (
      <section className="login-screen">
        <div className="auth-language">
          <LanguageSwitcher compact />
        </div>
        <Clock3 size={52} strokeWidth={2.2} />
        <h2>{t("auth.title")}</h2>
        <p>{t("auth.adminText")}</p>
        {error && <p className="error-banner">{error}</p>}
        <a className="google-login-link" href={loginURL}>
          <GoogleLogo />
          {t("auth.adminLogin")}
        </a>
      </section>
    );
  }

  return (
    <section className="login-screen">
      <div className="auth-language">
        <LanguageSwitcher compact />
      </div>
      <Clock3 size={52} strokeWidth={2.2} />
      <h2>{t("auth.title")}</h2>
      <p>{googleRegistration ? t("auth.googleRegistrationText") : mode === "login" ? t("auth.loginText") : t("auth.registerText")}</p>
      {noticeText && !admin && <p className="auth-notice">{noticeText}</p>}
      {(error || formError) && <p className="error-banner">{formError || error}</p>}

      <form className="auth-form" onSubmit={handleSubmit}>
        {googleRegistration && (
          <div className="google-registration-card">
            <GoogleLogo />
            <div>
              <span>{t("auth.googleConfirmed")}</span>
              <strong>{email}</strong>
            </div>
          </div>
        )}
        {mode === "register" && (
          <div className="auth-name-grid">
            <label>
              <span>{t("auth.lastName")}</span>
              <input value={lastName} onChange={(event) => setLastName(event.target.value)} required />
            </label>
            <label>
              <span>{t("auth.firstName")}</span>
              <input value={firstName} onChange={(event) => setFirstName(event.target.value)} required />
            </label>
            <label>
              <span>{t("auth.middleName")}</span>
              <input value={middleName} onChange={(event) => setMiddleName(event.target.value)} />
            </label>
          </div>
        )}
        {!googleRegistration && (
          <label>
            <span>{t("auth.email")}</span>
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              autoComplete="email"
              required
            />
          </label>
        )}
        {!googleRegistration && (
          <label>
            <span>{t("auth.password")}</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete={mode === "login" ? "current-password" : "new-password"}
              minLength={6}
              required
            />
          </label>
        )}
        {mode === "register" && !googleRegistration && (
          <label>
            <span>{t("auth.passwordRepeat")}</span>
            <input
              type="password"
              value={passwordRepeat}
              onChange={(event) => setPasswordRepeat(event.target.value)}
              autoComplete="new-password"
              minLength={6}
              required
            />
          </label>
        )}
        <button className="login-link" type="submit" disabled={submitting}>
          {submitting ? t("auth.submitting") : mode === "login" ? t("auth.loginSubmit") : t("auth.registerSubmit")}
        </button>
      </form>

      {!googleRegistration && (
        <>
          <button className="auth-switch" type="button" onClick={() => setMode(mode === "login" ? "register" : "login")}>
            {mode === "login" ? t("auth.switchToRegister") : t("auth.switchToLogin")}
          </button>
          <a className="google-login-link" href={loginURL}>
            <GoogleLogo />
            {t("auth.login")}
          </a>
        </>
      )}
    </section>
  );
}

function GoogleLogo() {
  return (
    <svg className="google-logo" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        fill="#4285F4"
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
      />
      <path
        fill="#34A853"
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C4 20.53 7.7 23 12 23z"
      />
      <path
        fill="#FBBC05"
        d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
      />
      <path
        fill="#EA4335"
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 4 3.47 2.18 7.07l3.66 2.84C6.71 7.31 9.14 5.38 12 5.38z"
      />
    </svg>
  );
}
