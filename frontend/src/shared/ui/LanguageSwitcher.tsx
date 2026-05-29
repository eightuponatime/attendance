import { useI18n } from "../i18n/i18n";

export function LanguageSwitcher({ compact = false }: { compact?: boolean }) {
  const { language, languages, setLanguage, t } = useI18n();

  return (
    <label className={`language-switcher ${compact ? "language-switcher-compact" : ""}`}>
      {!compact && <span>{t("profile.language")}</span>}
      <select
        aria-label={t("profile.language")}
        value={language}
        onChange={(event) => setLanguage(event.target.value as typeof language)}
      >
        {languages.map((item) => (
          <option key={item.code} value={item.code}>
            {item.flag} {compact ? item.code.toUpperCase() : item.label}
          </option>
        ))}
      </select>
    </label>
  );
}
