package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"attendance/config"
	"attendance/internal/domain"
	appMiddleware "attendance/internal/middleware"
	"attendance/internal/service"
	impl "attendance/internal/service/impl"

	"github.com/go-chi/chi/v5"
)

const (
	googleOAuthStateCookie    = "google_oauth_state"
	googleOAuthReturnToCookie = "google_oauth_return_to"
)

type AuthHandler struct {
	cfg             *config.Config
	usersService    service.UsersService
	sessionsService service.SessionsService
	adminService    service.AdminService
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

type localRegisterRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
}

type localLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthHandler(
	cfg *config.Config,
	usersService service.UsersService,
	sessionsService service.SessionsService,
	adminService service.AdminService,
) *AuthHandler {
	return &AuthHandler{
		cfg:             cfg,
		usersService:    usersService,
		sessionsService: sessionsService,
		adminService:    adminService,
	}
}

func (h *AuthHandler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/register", h.RegisterLocal)
	r.Post("/login", h.LoginLocal)
	r.Get("/google/login", h.GoogleLogin)
	r.Get("/google/callback", h.GoogleCallback)
	r.Get("/admin/google/login", h.AdminGoogleLogin)
	r.Get("/admin/google/callback", h.AdminGoogleCallback)
}

func (h *AuthHandler) RegisterLocal(w http.ResponseWriter, r *http.Request) {
	var request localRegisterRequest
	if err := readJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := h.usersService.RegisterLocal(r.Context(), domain.LocalRegisterInput{
		Email:      request.Email,
		Password:   request.Password,
		LastName:   request.LastName,
		FirstName:  request.FirstName,
		MiddleName: request.MiddleName,
	})
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	if !h.startUserSession(w, r, user) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}
	writeJSON(w, http.StatusOK, newUserResponse(user))
}

func (h *AuthHandler) LoginLocal(w http.ResponseWriter, r *http.Request) {
	var request localLoginRequest
	if err := readJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := h.usersService.LoginLocal(r.Context(), domain.LocalLoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		h.writeAuthError(w, err)
		return
	}

	if !h.startUserSession(w, r, user) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}
	writeJSON(w, http.StatusOK, newUserResponse(user))
}

func (h *AuthHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/logout", h.Logout)
}

func (h *AuthHandler) RegisterAdminRoutes(r chi.Router) {
	r.Post("/admin/logout", h.AdminLogout)
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.googleOAuthConfigured() {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "google oauth is not configured",
		})
		return
	}

	state, err := randomState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create oauth state",
		})
		return
	}

	http.SetCookie(w, h.newOAuthStateCookie(state))
	http.SetCookie(w, h.newOAuthReturnToCookie(r.URL.Query().Get("return_to")))
	http.Redirect(w, r, h.googleAuthURL(state, false), http.StatusFound)
}

func (h *AuthHandler) AdminGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.googleOAuthConfigured() {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "google oauth is not configured",
		})
		return
	}

	state, err := randomState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create oauth state",
		})
		return
	}

	http.SetCookie(w, h.newOAuthStateCookie(state))
	http.SetCookie(w, h.newOAuthReturnToCookie(r.URL.Query().Get("return_to")))
	http.Redirect(w, r, h.googleAuthURL(state, true), http.StatusFound)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if errMessage := r.URL.Query().Get("error"); errMessage != "" {
		h.redirectWithOAuthError(w, r, false, errMessage)
		return
	}

	if !h.validOAuthState(r) {
		h.redirectWithOAuthError(w, r, false, "Не удалось подтвердить вход через Google")
		return
	}
	http.SetCookie(w, h.expiredOAuthStateCookie())
	redirectURL := h.frontendRedirectURL(r, false)
	http.SetCookie(w, h.expiredOAuthReturnToCookie())

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		h.redirectWithOAuthError(w, r, false, "Google не вернул код авторизации")
		return
	}

	userInfo, err := h.googleUserInfo(r.Context(), code, false)
	if err != nil {
		h.redirectWithOAuthError(w, r, false, "Не удалось получить профиль Google")
		return
	}
	if !userInfo.EmailVerified {
		h.redirectWithOAuthError(w, r, false, "Email Google не подтвержден")
		return
	}

	user, err := h.usersService.FindOrCreateFromGoogle(r.Context(), domain.GoogleUserInput{
		GoogleSub: userInfo.Sub,
		Email:     userInfo.Email,
		FullName:  userInfo.Name,
	})
	if err != nil {
		if errors.Is(err, impl.ErrInvalidCredentials) {
			h.redirectWithOAuthError(w, r, false, "Сначала зарегистрируйтесь по email и паролю")
			return
		}
		h.redirectWithOAuthError(w, r, false, "Не удалось войти через Google")
		return
	}

	if !h.startUserSession(w, r, user) {
		h.redirectWithOAuthError(w, r, false, "Не удалось создать сессию")
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *AuthHandler) AdminGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if errMessage := r.URL.Query().Get("error"); errMessage != "" {
		h.redirectWithOAuthError(w, r, true, errMessage)
		return
	}

	if !h.validOAuthState(r) {
		h.redirectWithOAuthError(w, r, true, "Не удалось подтвердить вход через Google")
		return
	}
	http.SetCookie(w, h.expiredOAuthStateCookie())
	redirectURL := h.frontendRedirectURL(r, true)
	http.SetCookie(w, h.expiredOAuthReturnToCookie())

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		h.redirectWithOAuthError(w, r, true, "Google не вернул код авторизации")
		return
	}

	userInfo, err := h.googleUserInfo(r.Context(), code, true)
	if err != nil {
		log.Printf("admin google oauth: failed to get profile: %v", err)
		h.redirectWithOAuthError(w, r, true, "Не удалось получить профиль Google")
		return
	}
	log.Printf(
		"admin google oauth: userinfo sub=%q email=%q email_verified=%t name=%q",
		userInfo.Sub,
		userInfo.Email,
		userInfo.EmailVerified,
		userInfo.Name,
	)
	if !userInfo.EmailVerified {
		log.Printf("admin google oauth: email is not verified: email=%q", userInfo.Email)
		h.redirectWithOAuthError(w, r, true, "Email Google не подтвержден")
		return
	}

	session, err := h.adminService.CreateSession(r.Context(), domain.GoogleUserInput{
		GoogleSub: userInfo.Sub,
		Email:     userInfo.Email,
		FullName:  userInfo.Name,
	})
	if err != nil {
		log.Printf("admin google oauth: failed to create admin session for email=%q: %v", userInfo.Email, err)
		h.redirectWithOAuthError(w, r, true, "Этот Google аккаунт не имеет доступа к админ-панели")
		return
	}

	http.SetCookie(w, appMiddleware.NewAdminSessionCookie(h.cfg, session.Id))
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := appMiddleware.SessionIDFromContext(r.Context())
	if ok {
		_ = h.sessionsService.Revoke(r.Context(), sessionID)
	}

	http.SetCookie(w, appMiddleware.NewExpiredSessionCookie(h.cfg))
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *AuthHandler) AdminLogout(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := appMiddleware.AdminSessionIDFromContext(r.Context())
	if ok {
		_ = h.adminService.RevokeSession(r.Context(), sessionID)
	}

	http.SetCookie(w, appMiddleware.NewExpiredAdminSessionCookie(h.cfg))
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *AuthHandler) startUserSession(w http.ResponseWriter, r *http.Request, user *domain.Users) bool {
	session, err := h.sessionsService.Create(r.Context(), user.Id)
	if err != nil {
		return false
	}

	http.SetCookie(w, appMiddleware.NewSessionCookie(h.cfg, session.Id))
	return true
}

func (h *AuthHandler) writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, impl.ErrEmailAlreadyExists):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
	case errors.Is(err, impl.ErrInvalidCredentials):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	case errors.Is(err, impl.ErrInvalidLocalAuth):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "auth failed"})
	}
}

func (h *AuthHandler) googleAuthURL(state string, admin bool) string {
	query := url.Values{}
	query.Set("client_id", h.cfg.GoogleClientID)
	query.Set("redirect_uri", h.googleRedirectURL(admin))
	query.Set("response_type", "code")
	query.Set("scope", "openid email profile")
	query.Set("state", state)
	query.Set("prompt", "select_account")

	return "https://accounts.google.com/o/oauth2/v2/auth?" + query.Encode()
}

func (h *AuthHandler) googleRedirectURL(admin bool) string {
	if !admin {
		return h.cfg.GoogleRedirectURL
	}

	return h.cfg.AdminRedirectURL()
}

func (h *AuthHandler) googleUserInfo(ctx context.Context, code string, admin bool) (*googleUserInfo, error) {
	token, err := h.exchangeCode(ctx, code, admin)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://openidconnect.googleapis.com/v1/userinfo",
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("google userinfo: status=%s", resp.Status)
		return nil, fmt.Errorf("google userinfo status: %s", resp.Status)
	}

	var userInfo googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}
	if userInfo.Sub == "" || userInfo.Email == "" || userInfo.Name == "" {
		return nil, errors.New("google profile is incomplete")
	}

	return &userInfo, nil
}

func (h *AuthHandler) exchangeCode(ctx context.Context, code string, admin bool) (*googleTokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", h.cfg.GoogleClientID)
	form.Set("client_secret", h.cfg.GoogleSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", h.googleRedirectURL(admin))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://oauth2.googleapis.com/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var token googleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf(
			"google token exchange: status=%s error=%q description=%q",
			resp.Status,
			token.Error,
			token.Description,
		)
		return nil, fmt.Errorf("google token status: %s: %s", resp.Status, token.Description)
	}
	if token.AccessToken == "" {
		return nil, errors.New("google access token is empty")
	}

	return &token, nil
}

func (h *AuthHandler) googleOAuthConfigured() bool {
	return h.cfg.GoogleClientID != "" &&
		h.cfg.GoogleSecret != "" &&
		h.cfg.GoogleRedirectURL != ""
}

func (h *AuthHandler) validOAuthState(r *http.Request) bool {
	cookie, err := r.Cookie(googleOAuthStateCookie)
	if err != nil || cookie.Value == "" {
		return false
	}

	return r.URL.Query().Get("state") == cookie.Value
}

func (h *AuthHandler) newOAuthStateCookie(state string) *http.Cookie {
	return &http.Cookie{
		Name:     googleOAuthStateCookie,
		Value:    state,
		Path:     "/auth",
		MaxAge:   int((10 * time.Minute).Seconds()),
		Expires:  time.Now().UTC().Add(10 * time.Minute),
		HttpOnly: true,
		Secure:   h.cfg.Env != "development",
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *AuthHandler) expiredOAuthStateCookie() *http.Cookie {
	return &http.Cookie{
		Name:     googleOAuthStateCookie,
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   h.cfg.Env != "development",
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *AuthHandler) newOAuthReturnToCookie(returnTo string) *http.Cookie {
	return &http.Cookie{
		Name:     googleOAuthReturnToCookie,
		Value:    safeReturnTo(returnTo),
		Path:     "/auth",
		MaxAge:   int((10 * time.Minute).Seconds()),
		Expires:  time.Now().UTC().Add(10 * time.Minute),
		HttpOnly: true,
		Secure:   h.cfg.Env != "development",
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *AuthHandler) expiredOAuthReturnToCookie() *http.Cookie {
	return &http.Cookie{
		Name:     googleOAuthReturnToCookie,
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   h.cfg.Env != "development",
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *AuthHandler) frontendRedirectURL(r *http.Request, admin bool) string {
	baseURL := h.cfg.FrontendURL
	if admin {
		baseURL = h.cfg.AdminFrontendURLValue()
	}

	cookie, err := r.Cookie(googleOAuthReturnToCookie)
	if err != nil || cookie.Value == "" {
		return baseURL
	}

	return strings.TrimRight(baseURL, "/") + safeReturnTo(cookie.Value)
}

func (h *AuthHandler) redirectWithOAuthError(w http.ResponseWriter, r *http.Request, admin bool, message string) {
	redirectURL := h.frontendRedirectURL(r, admin)
	parsed, err := url.Parse(redirectURL)
	if err == nil {
		query := parsed.Query()
		query.Set("auth_error", message)
		parsed.RawQuery = query.Encode()
		redirectURL = parsed.String()
	}

	http.SetCookie(w, h.expiredOAuthStateCookie())
	http.SetCookie(w, h.expiredOAuthReturnToCookie())
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func safeReturnTo(returnTo string) string {
	trimmed := strings.TrimSpace(returnTo)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "//") {
		return "/"
	}

	return trimmed
}

func randomState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
