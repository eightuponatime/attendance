import { BarChart3, CalendarDays, Clock3, UserRound } from "lucide-react";
import type { ReactNode } from "react";
import { useI18n } from "../shared/i18n/i18n";

export type Tab = "home" | "stats" | "calendar" | "profile";

export function BottomNav({
  active,
  onChange,
}: {
  active: Tab;
  onChange: (tab: Tab) => void;
}) {
  const { t } = useI18n();
  return (
    <nav className="bottom-nav" aria-label="Main navigation">
      <NavButton
        active={active === "home"}
        icon={<Clock3 size={30} />}
        label={t("nav.home")}
        onClick={() => onChange("home")}
      />
      <NavButton
        active={active === "stats"}
        icon={<BarChart3 size={30} />}
        label={t("nav.stats")}
        onClick={() => onChange("stats")}
      />
      <NavButton
        active={active === "calendar"}
        icon={<CalendarDays size={30} />}
        label={t("nav.calendar")}
        onClick={() => onChange("calendar")}
      />
      <NavButton
        active={active === "profile"}
        icon={<UserRound size={30} />}
        label={t("nav.profile")}
        onClick={() => onChange("profile")}
      />
    </nav>
  );
}

function NavButton({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  icon: ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      className={`nav-button ${active ? "nav-button-active" : ""}`}
      type="button"
      onClick={onClick}
    >
      {icon}
      <span>{label}</span>
    </button>
  );
}
