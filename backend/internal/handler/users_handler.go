package handler

import (
	"net/http"

	appMiddleware "attendance/internal/middleware"
	"attendance/internal/service"

	"github.com/go-chi/chi/v5"
)

type UsersHandler struct {
	usersService service.UsersService
}

func NewUsersHandler(usersService service.UsersService) *UsersHandler {
	return &UsersHandler{
		usersService: usersService,
	}
}

func (h *UsersHandler) RegisterRoutes(r chi.Router) {
	r.Get("/me", h.Me)
}

func (h *UsersHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := appMiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	user, err := h.usersService.GetByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get user",
		})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	writeJSON(w, http.StatusOK, newUserResponse(user))
}
