package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"attendance/internal/service"

	"github.com/google/uuid"
)

const (
	SessionCookieName      = "session_id"
	AdminSessionCookieName = "admin_session_id"
)

type contextKey string

const (
	sessionIDContextKey contextKey = "session_id"
	userIDContextKey    contextKey = "user_id"
	adminSessionIDKey   contextKey = "admin_session_id"
	adminEmailKey       contextKey = "admin_email"
	adminFullNameKey    contextKey = "admin_full_name"
)

type AuthMiddleware struct {
	sessionsService service.SessionsService
	adminService    service.AdminService
}

func NewAuthMiddleware(
	sessionsService service.SessionsService,
	adminService service.AdminService,
) *AuthMiddleware {
	return &AuthMiddleware{
		sessionsService: sessionsService,
		adminService:    adminService,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			writeUnauthorized(w)
			return
		}

		sessionID, err := uuid.Parse(cookie.Value)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		session, err := m.sessionsService.GetValidByID(r.Context(), sessionID)
		if err != nil || session == nil {
			writeUnauthorized(w)
			return
		}

		ctx := context.WithValue(r.Context(), sessionIDContextKey, session.Id)
		ctx = context.WithValue(ctx, userIDContextKey, session.UserId)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(AdminSessionCookieName)
		if err != nil || cookie.Value == "" {
			writeUnauthorized(w)
			return
		}

		sessionID, err := uuid.Parse(cookie.Value)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		session, err := m.adminService.GetValidSessionByID(r.Context(), sessionID)
		if err != nil {
			writeForbidden(w)
			return
		}
		if session == nil {
			writeUnauthorized(w)
			return
		}

		ctx := context.WithValue(r.Context(), adminSessionIDKey, session.Id)
		ctx = context.WithValue(ctx, adminEmailKey, session.Email)
		ctx = context.WithValue(ctx, adminFullNameKey, session.FullName)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func SessionIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	sessionID, ok := ctx.Value(sessionIDContextKey).(uuid.UUID)
	return sessionID, ok && sessionID != uuid.Nil
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return userID, ok && userID != uuid.Nil
}

func AdminSessionIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	sessionID, ok := ctx.Value(adminSessionIDKey).(uuid.UUID)
	return sessionID, ok && sessionID != uuid.Nil
}

func AdminEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(adminEmailKey).(string)
	return email, ok && email != ""
}

func AdminFullNameFromContext(ctx context.Context) (string, bool) {
	fullName, ok := ctx.Value(adminFullNameKey).(string)
	return fullName, ok && fullName != ""
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "unauthorized",
	})
}

func writeForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "forbidden",
	})
}
