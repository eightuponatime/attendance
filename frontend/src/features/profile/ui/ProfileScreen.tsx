import { UserRound } from "lucide-react";
import { useI18n } from "../../../shared/i18n/i18n";
import type { User } from "../../../shared/types/api";

export function ProfileScreen({ user, onLogout }: { user: User; onLogout: () => void }) {
  const { t } = useI18n();

  return (
    <section className="screen-content">
      <div className="profile-card">
        <div className="profile-avatar">
          <UserRound size={36} />
        </div>
        <strong>{user.full_name}</strong>
        <span>{user.email}</span>

        <button className="secondary-button" type="button" onClick={onLogout}>
          {t("profile.logout")}
        </button>
      </div>
    </section>
  );
}
