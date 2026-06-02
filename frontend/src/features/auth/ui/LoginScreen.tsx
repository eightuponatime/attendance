import { useState } from "react";
import type { FormEvent } from "react";
import { Clock3 } from "lucide-react";
import { login, register } from "../api/authApi";
import { useI18n } from "../../../shared/i18n/i18n";

export function LoginScreen({
  error,
  admin = false,
  onAuthenticated,
}: {
  error: string | null;
  admin?: boolean;
  onAuthenticated?: () => void;
}) {
  const { t } = useI18n();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordRepeat, setPasswordRepeat] = useState("");
  const [lastName, setLastName] = useState("");
  const [firstName, setFirstName] = useState("");
  const [middleName, setMiddleName] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const loginPath = admin ? "/auth/admin/google/login" : "/auth/google/login";
  const loginURL = `${loginPath}?${new URLSearchParams({
    return_to: `${window.location.pathname}${window.location.search}`,
  })}`;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);

    if (mode === "register" && password !== passwordRepeat) {
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
          password,
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
        <Clock3 size={52} strokeWidth={2.2} />
        <h2>{t("auth.title")}</h2>
        <p>{t("auth.adminText")}</p>
        {error && <p className="error-banner">{error}</p>}
        <a className="login-link" href={loginURL}>
          {t("auth.adminLogin")}
        </a>
      </section>
    );
  }

  return (
    <section className="login-screen">
      <Clock3 size={52} strokeWidth={2.2} />
      <h2>{t("auth.title")}</h2>
      <p>{mode === "login" ? t("auth.loginText") : t("auth.registerText")}</p>
      {(error || formError) && <p className="error-banner">{formError || error}</p>}

      <form className="auth-form" onSubmit={handleSubmit}>
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
        {mode === "register" && (
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

      <button className="auth-switch" type="button" onClick={() => setMode(mode === "login" ? "register" : "login")}>
        {mode === "login" ? t("auth.switchToRegister") : t("auth.switchToLogin")}
      </button>
      <a className="auth-google-link" href={loginURL}>
        {t("auth.login")}
      </a>
    </section>
  );
}
