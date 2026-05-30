package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	DatabaseURL            string
	Port                   string
	Env                    string
	BusinessTimezone       string
	CorporateDomain        string
	SessionTTL             time.Duration
	FrontendURL            string
	AdminFrontendURL       string
	GoogleClientID         string
	GoogleSecret           string
	GoogleRedirectURL      string
	GoogleAdminRedirectURL string
	WorkdayStart           string
	WorkdayEnd             string
	SMTPHost               string
	SMTPPort               string
	SMTPUser               string
	SMTPPassword           string
	SMTPFrom               string
	SMTPStartTLS           bool
	ReportEmailEnabled     bool
	ReportEmailTime        string
	HeartbeatInterval      time.Duration
	OutageThreshold        time.Duration
	OutageImpactStart      string
	OutageImpactEnd        string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file")
	}

	return &Config{
		DatabaseURL:            getEnv("DATABASE_URL", ""),
		Port:                   getEnv("PORT", "8080"),
		Env:                    getEnv("ENV", "development"),
		BusinessTimezone:       getEnv("BUSINESS_TIMEZONE", "Asia/Almaty"),
		CorporateDomain:        getEnv("CORPORATE_DOMAIN", ""),
		SessionTTL:             getDurationEnv("SESSION_TTL", 30*24*time.Hour),
		FrontendURL:            getEnv("FRONTEND_URL", "/"),
		AdminFrontendURL:       getEnv("ADMIN_FRONTEND_URL", ""),
		GoogleClientID:         getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleSecret:           getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:      getEnv("GOOGLE_REDIRECT_URL", ""),
		GoogleAdminRedirectURL: getEnv("GOOGLE_ADMIN_REDIRECT_URL", ""),
		WorkdayStart:           getEnv("WORKDAY_START", "08:00"),
		WorkdayEnd:             getEnv("WORKDAY_END", "17:00"),
		SMTPHost:               getEnv("SMTP_HOST", ""),
		SMTPPort:               getEnv("SMTP_PORT", "587"),
		SMTPUser:               getEnv("SMTP_USER", getEnv("SMTP_USERNAME", "")),
		SMTPPassword:           getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:               getEnv("SMTP_FROM", getEnv("SMTP_USERNAME", "")),
		SMTPStartTLS:           getBoolEnv("SMTP_STARTTLS", true),
		ReportEmailEnabled:     getBoolEnv("REPORT_EMAIL_ENABLED", false),
		ReportEmailTime:        getEnv("REPORT_EMAIL_TIME", "08:30"),
		HeartbeatInterval:      getDurationEnv("HEARTBEAT_INTERVAL", time.Minute),
		OutageThreshold:        getDurationEnv("OUTAGE_THRESHOLD", 5*time.Minute),
		OutageImpactStart:      getEnv("OUTAGE_IMPACT_START", "06:00"),
		OutageImpactEnd:        getEnv("OUTAGE_IMPACT_END", "19:00"),
	}, nil
}

func getEnv(key string, fallback string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func getBoolEnv(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func (c *Config) LogConfig(logger *zap.SugaredLogger) {
	logger.Infow(
		"config loaded",
		"port", c.Port,
		"env", c.Env,
		"BusinessTimezone", c.BusinessTimezone,
		"corporateDomain", c.CorporateDomain,
		"sessionTTL", c.SessionTTL.String(),
		"frontendURL", c.FrontendURL,
		"adminFrontendURL", c.AdminFrontendURLValue(),
		"googleOAuthConfigured", c.GoogleClientID != "" && c.GoogleSecret != "" && c.GoogleRedirectURL != "",
		"googleAdminOAuthConfigured", c.GoogleClientID != "" && c.GoogleSecret != "" && c.AdminRedirectURL() != "",
		"workdayStart", c.WorkdayStart,
		"workdayEnd", c.WorkdayEnd,
		"smtpConfigured", c.SMTPConfigured(),
		"reportEmailEnabled", c.ReportEmailEnabled,
		"reportEmailTime", c.ReportEmailTime,
		"heartbeatInterval", c.HeartbeatInterval.String(),
		"outageThreshold", c.OutageThreshold.String(),
		"outageImpactStart", c.OutageImpactStart,
		"outageImpactEnd", c.OutageImpactEnd,
	)
}

func (c *Config) AdminRedirectURL() string {
	if c.GoogleAdminRedirectURL != "" {
		return c.GoogleAdminRedirectURL
	}

	return strings.Replace(c.GoogleRedirectURL, "/auth/google/callback", "/auth/admin/google/callback", 1)
}

func (c *Config) AdminFrontendURLValue() string {
	if c.AdminFrontendURL != "" {
		return c.AdminFrontendURL
	}

	adminRedirectURL := c.AdminRedirectURL()
	if strings.HasPrefix(adminRedirectURL, "http://") || strings.HasPrefix(adminRedirectURL, "https://") {
		if index := strings.Index(adminRedirectURL, "/auth/"); index > 0 {
			return adminRedirectURL[:index]
		}
	}

	return c.FrontendURL
}

func (c *Config) SMTPConfigured() bool {
	return c.SMTPHost != "" && c.SMTPPort != "" && c.SMTPFrom != ""
}
