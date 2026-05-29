package middleware

import (
	"net/http"
	"time"

	"attendance/config"

	"github.com/google/uuid"
)

func NewSessionCookie(cfg *config.Config, sessionID uuid.UUID) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID.String(),
		Path:     "/",
		MaxAge:   int(cfg.SessionTTL.Seconds()),
		Expires:  time.Now().UTC().Add(cfg.SessionTTL),
		HttpOnly: true,
		Secure:   cfg.Env != "development",
		SameSite: http.SameSiteLaxMode,
	}
}

func NewExpiredSessionCookie(cfg *config.Config) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   cfg.Env != "development",
		SameSite: http.SameSiteLaxMode,
	}
}

func NewAdminSessionCookie(cfg *config.Config, sessionID uuid.UUID) *http.Cookie {
	return &http.Cookie{
		Name:     AdminSessionCookieName,
		Value:    sessionID.String(),
		Path:     "/",
		MaxAge:   int(cfg.SessionTTL.Seconds()),
		Expires:  time.Now().UTC().Add(cfg.SessionTTL),
		HttpOnly: true,
		Secure:   cfg.Env != "development",
		SameSite: http.SameSiteLaxMode,
	}
}

func NewExpiredAdminSessionCookie(cfg *config.Config) *http.Cookie {
	return &http.Cookie{
		Name:     AdminSessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   cfg.Env != "development",
		SameSite: http.SameSiteLaxMode,
	}
}
