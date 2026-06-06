package handler

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"attendance/internal/domain"
	appMiddleware "attendance/internal/middleware"
	"attendance/internal/service"
	"attendance/internal/service/impl"

	"github.com/go-chi/chi/v5"
)

type AttendanceHandler struct {
	attendanceService service.AttendanceService
}

type attendanceMarkRequest struct {
	PhoneModel string `json:"phone_model"`
	Browser    string `json:"browser"`
	DeviceId   string `json:"device_id"`
}

type attendanceExplanationRequest struct {
	BusinessDate string `json:"business_date"`
	ReasonType   string `json:"reason_type"`
	Comment      string `json:"comment"`
}

func NewAttendanceHandler(attendanceService service.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{attendanceService: attendanceService}
}

func (h *AttendanceHandler) RegisterRoutes(r chi.Router) {
	r.Get("/attendance/today", h.Today)
	r.Get("/attendance/summary", h.Summary)
	r.Post("/attendance/check-in", h.CheckIn)
	r.Post("/attendance/check-out", h.CheckOut)
	r.Post("/attendance/explanations", h.SubmitExplanation)
}

func (h *AttendanceHandler) Today(w http.ResponseWriter, r *http.Request) {
	userID, ok := appMiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	today, err := h.attendanceService.Today(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get attendance"})
		return
	}

	writeJSON(w, http.StatusOK, newAttendanceTodayResponse(today))
}

func (h *AttendanceHandler) Summary(w http.ResponseWriter, r *http.Request) {
	userID, ok := appMiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	from, to, err := summaryRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid summary range"})
		return
	}

	summary, err := h.attendanceService.Summary(r.Context(), userID, from, to)
	if err != nil {
		h.writeAttendanceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAttendanceSummaryResponse(summary))
}

func (h *AttendanceHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	h.mark(w, r, h.attendanceService.CheckIn)
}

func (h *AttendanceHandler) CheckOut(w http.ResponseWriter, r *http.Request) {
	h.mark(w, r, h.attendanceService.CheckOut)
}

func (h *AttendanceHandler) SubmitExplanation(w http.ResponseWriter, r *http.Request) {
	userID, ok := appMiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var request attendanceExplanationRequest
	if err := readJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	businessDate, err := time.Parse("2006-01-02", request.BusinessDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid business_date"})
		return
	}

	explanation, err := h.attendanceService.SubmitExplanation(r.Context(), domain.CreateAttendanceExplanationInput{
		UserId:       userID,
		BusinessDate: businessDate,
		ReasonType:   request.ReasonType,
		Comment:      request.Comment,
	})
	if err != nil {
		h.writeAttendanceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAttendanceExplanationResponse(*explanation))
}

func (h *AttendanceHandler) mark(
	w http.ResponseWriter,
	r *http.Request,
	markFunc func(context.Context, domain.AttendanceMarkInput) (*domain.AttendanceToday, error),
) {
	userID, ok := appMiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var request attendanceMarkRequest
	if err := readJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	today, err := markFunc(r.Context(), domain.AttendanceMarkInput{
		UserId:     userID,
		PhoneModel: request.PhoneModel,
		Browser:    request.Browser,
		DeviceId:   request.DeviceId,
		ExternalIp: externalIP(r),
	})
	if err != nil {
		h.writeAttendanceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAttendanceTodayResponse(today))
}

func (h *AttendanceHandler) writeAttendanceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, impl.ErrInvalidAttendanceInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, impl.ErrCheckInRequired):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "check-in is required before check-out"})
	case errors.Is(err, impl.ErrAttendanceAlreadyDone):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "attendance event already exists"})
	case errors.Is(err, impl.ErrInvalidAttendanceRange):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, impl.ErrExplanationUnavailable):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "explanation is not available for this day"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to mark attendance"})
	}
}

func summaryRange(r *http.Request) (time.Time, time.Time, error) {
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return from, to, nil
}

func externalIP(r *http.Request) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
