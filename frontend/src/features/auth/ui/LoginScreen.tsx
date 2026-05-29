import { Clock3 } from "lucide-react";
import { useI18n } from "../../../shared/i18n/i18n";

export function LoginScreen({
  error,
  admin = false,
}: {
  error: string | null;
  admin?: boolean;
}) {
  const { t } = useI18n();
  const loginPath = admin ? "/auth/admin/google/login" : "/auth/google/login";
  const loginURL = `${loginPath}?${new URLSearchParams({
    return_to: `${window.location.pathname}${window.location.search}`,
  })}`;

  return (
    <section className="login-screen">
      <Clock3 size={52} strokeWidth={2.2} />
      <h2>{t("auth.title")}</h2>
      <p>{t("auth.text")}</p>
      {error && <p className="error-banner">{error}</p>}
      <a className="login-link" href={loginURL}>
        {admin ? t("auth.adminLogin") : t("auth.login")}
      </a>
    </section>
  );
}
