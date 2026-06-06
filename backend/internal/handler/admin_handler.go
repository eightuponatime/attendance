package handler

import (
	"errors"
	"net/http"
	"time"

	"attendance/internal/domain"
	"attendance/internal/mailer"
	appMiddleware "attendance/internal/middleware"
	"attendance/internal/service"
	"attendance/internal/service/impl"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	adminService service.AdminService
	reportMailer *mailer.ReportMailer
}

type adminAccessRequest struct {
	Email string `json:"email"`
}

type adminOutageRepairRequest struct {
	ResolutionNote string                         `json:"resolution_note"`
	Items          []adminOutageRepairItemRequest `json:"items"`
}

type adminOutageRepairItemRequest struct {
	UserId     string  `json:"user_id"`
	CheckInAt  *string `json:"check_in_at"`
	CheckOutAt *string `json:"check_out_at"`
}

type adminExplanationDecisionRequest struct {
	ReviewNote string  `json:"review_note"`
	CheckInAt  *string `json:"check_in_at"`
	CheckOutAt *string `json:"check_out_at"`
}

func NewAdminHandler(adminService service.AdminService, reportMailer *mailer.ReportMailer) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
		reportMailer: reportMailer,
	}
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/me", h.Me)
	r.Get("/admin/access", h.ListAccess)
	r.Post("/admin/access", h.AddAccess)
	r.Delete("/admin/access/{email}", h.RevokeAccess)
	r.Get("/admin/reports", h.ListReports)
	r.Post("/admin/reports/email", h.SendEmailReport)
	r.Get("/admin/sessions", h.ListSessions)
	r.Post("/admin/sessions/{sessionID}/revoke", h.RevokeSession)
	r.Get("/admin/employees", h.EmployeesMonth)
	r.Get("/admin/employees/{userID}", h.EmployeeMonth)
	r.Get("/admin/suspicious-activity", h.SuspiciousActivity)
	r.Get("/admin/system-outages", h.SystemOutages)
	r.Get("/admin/system-outages/{outageID}/day", h.SystemOutageDay)
	r.Post("/admin/system-outages/{outageID}/repair", h.RepairSystemOutage)
	r.Get("/admin/explanations", h.ListExplanations)
	r.Post("/admin/explanations/{explanationID}/approve", h.ApproveExplanation)
	r.Post("/admin/explanations/{explanationID}/reject", h.RejectExplanation)
}

func (h *AdminHandler) Me(w http.ResponseWriter, r *http.Request) {
	email, ok := appMiddleware.AdminEmailFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	fullName, _ := appMiddleware.AdminFullNameFromContext(r.Context())

	isAdmin, err := h.adminService.IsAdmin(r.Context(), email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to check admin access"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"is_admin":  isAdmin,
		"email":     email,
		"full_name": fullName,
	})
}

func (h *AdminHandler) ListAccess(w http.ResponseWriter, r *http.Request) {
	rows, err := h.adminService.ListAccess(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list admin access"})
		return
	}

	writeJSON(w, http.StatusOK, newAdminAccessListResponse(rows))
}

func (h *AdminHandler) AddAccess(w http.ResponseWriter, r *http.Request) {
	email, ok := appMiddleware.AdminEmailFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var request adminAccessRequest
	if err := readJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	row, err := h.adminService.AddAccess(r.Context(), domain.CreateAdminAccessInput{
		Email:     request.Email,
		CreatedBy: email,
	})
	if err != nil {
		h.writeAdminError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newAdminAccessResponse(row))
}

func (h *AdminHandler) RevokeAccess(w http.ResponseWriter, r *http.Request) {
	email, ok := appMiddleware.AdminEmailFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.adminService.RevokeAccess(r.Context(), email, chi.URLParam(r, "email")); err != nil {
		h.writeAdminError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	rows, err := h.adminService.ListReports(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list reports"})
		return
	}

	writeJSON(w, http.StatusOK, newAdminReportListResponse(rows))
}

func (h *AdminHandler) SendEmailReport(w http.ResponseWriter, r *http.Request) {
	if h.reportMailer == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "report mailer is not configured"})
		return
	}

	if err := h.reportMailer.SendCurrentMonthReport(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to send report email"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *AdminHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	rows, err := h.adminService.ListSessions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list admin sessions"})
		return
	}

	writeJSON(w, http.StatusOK, newAdminSessionListResponse(rows))
}

func (h *AdminHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session id"})
		return
	}

	if err := h.adminService.RevokeSession(r.Context(), sessionID); err != nil {
		h.writeAdminError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) EmployeesMonth(w http.ResponseWriter, r *http.Request) {
	from, to, err := adminMonthRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid month"})
		return
	}

	overview, err := h.adminService.EmployeesMonth(r.Context(), from, to)
	if err != nil {
		h.writeAdminError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAdminEmployeesMonthResponse(overview))
}

func (h *AdminHandler) EmployeeMonth(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	from, to, err := adminMonthRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid month"})
		return
	}

	summary, err := h.adminService.EmployeeMonth(r.Context(), userID, from, to)
	if err != nil {
		h.writeAdminError(w, err)
		return
	}
	if summary == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "employee not found"})
		return
	}

	writeJSON(w, http.StatusOK, newAdminEmployeeMonthDetailResponse(summary))
}

func (h *AdminHandler) SuspiciousActivity(w http.ResponseWriter, r *http.Request) {
	from, to, err := adminMonthRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid month"})
		return
	}

	activity, err := h.adminService.SuspiciousActivity(r.Context(), from, to)
	if err != nil {
		h.writeAdminError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAdminSuspiciousActivityResponse(activity))
}

func (h *AdminHandler) SystemOutages(w http.ResponseWriter, r *http.Request) {
	from, to, err := adminMonthRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid month"})
		return
	}

	outages, err := h.adminService.ListSystemOutages(r.Context(), from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list system outages"})
		return
	}

	writeJSON(w, http.StatusOK, newAdminSystemOutageListResponse(outages))
}

func (h *AdminHandler) SystemOutageDay(w http.ResponseWriter, r *http.Request) {
	outageID, err := uuid.Parse(chi.URLParam(r, "outageID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid outage id"})
		return
	}

	outage, rows, err := h.adminService.OutageDayEmployees(r.Context(), outageID)
	if err != nil {
		h.writeAdminError(w, err)
		return
	}
	if outage == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "outage not found"})
		return
	}

	writeJSON(w, http.StatusOK, newAdminOutageDayResponse(outage, rows))
}

func (h *AdminHandler) RepairSystemOutage(w http.ResponseWriter, r *http.Request) {
	adminEmail, ok := appMiddleware.AdminEmailFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	outageID, err := uuid.Parse(chi.URLParam(r, "outageID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid outage id"})
		return
	}

	var request adminOutageRepairRequest
	if err := readJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	input, err := h.newOutageRepairInput(outageID, adminEmail, request)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := h.adminService.RepairOutageDay(r.Context(), input); err != nil {
		h.writeAdminError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (h *AdminHandler) ListExplanations(w http.ResponseWriter, r *http.Request) {
	from, to, err := adminMonthRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid month"})
		return
	}

	rows, err := h.adminService.ListExplanations(r.Context(), from, to, r.URL.Query().Get("status"))
	if err != nil {
		h.writeAdminError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAdminExplanationListResponse(rows))
}

func (h *AdminHandler) ApproveExplanation(w http.ResponseWriter, r *http.Request) {
	h.reviewExplanation(w, r, true)
}

func (h *AdminHandler) RejectExplanation(w http.ResponseWriter, r *http.Request) {
	h.reviewExplanation(w, r, false)
}

func (h *AdminHandler) reviewExplanation(w http.ResponseWriter, r *http.Request, approve bool) {
	adminEmail, ok := appMiddleware.AdminEmailFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	explanationID, err := uuid.Parse(chi.URLParam(r, "explanationID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid explanation id"})
		return
	}

	var request adminExplanationDecisionRequest
	if err := readJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	input, err := h.newExplanationDecisionInput(explanationID, adminEmail, request)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if approve {
		err = h.adminService.ApproveExplanation(r.Context(), input)
	} else {
		err = h.adminService.RejectExplanation(r.Context(), input)
	}
	if err != nil {
		h.writeAdminError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (h *AdminHandler) writeAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, impl.ErrInvalidAdminInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, impl.ErrCannotRevokeSelf):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "cannot revoke your own admin access"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "admin operation failed"})
	}
}

func (h *AdminHandler) newOutageRepairInput(
	outageID uuid.UUID,
	adminEmail string,
	request adminOutageRepairRequest,
) (domain.AdminOutageRepairInput, error) {
	items := make([]domain.AdminOutageRepairItem, 0, len(request.Items))
	for _, row := range request.Items {
		userID, err := uuid.Parse(row.UserId)
		if err != nil {
			return domain.AdminOutageRepairInput{}, err
		}

		item := domain.AdminOutageRepairItem{UserId: userID}
		if row.CheckInAt != nil && *row.CheckInAt != "" {
			value, err := parseRepairClock(*row.CheckInAt)
			if err != nil {
				return domain.AdminOutageRepairInput{}, err
			}
			item.CheckInAt = &value
		}
		if row.CheckOutAt != nil && *row.CheckOutAt != "" {
			value, err := parseRepairClock(*row.CheckOutAt)
			if err != nil {
				return domain.AdminOutageRepairInput{}, err
			}
			item.CheckOutAt = &value
		}
		if item.CheckInAt == nil && item.CheckOutAt == nil {
			continue
		}
		items = append(items, item)
	}

	return domain.AdminOutageRepairInput{
		OutageId:       outageID,
		AdminEmail:     adminEmail,
		ResolutionNote: request.ResolutionNote,
		Items:          items,
	}, nil
}

func (h *AdminHandler) newExplanationDecisionInput(
	explanationID uuid.UUID,
	adminEmail string,
	request adminExplanationDecisionRequest,
) (domain.AdminExplanationDecisionInput, error) {
	input := domain.AdminExplanationDecisionInput{
		ExplanationId: explanationID,
		AdminEmail:    adminEmail,
		ReviewNote:    request.ReviewNote,
	}
	if request.CheckInAt != nil && *request.CheckInAt != "" {
		value, err := parseRepairClock(*request.CheckInAt)
		if err != nil {
			return domain.AdminExplanationDecisionInput{}, err
		}
		input.CheckInAt = &value
	}
	if request.CheckOutAt != nil && *request.CheckOutAt != "" {
		value, err := parseRepairClock(*request.CheckOutAt)
		if err != nil {
			return domain.AdminExplanationDecisionInput{}, err
		}
		input.CheckOutAt = &value
	}

	return input, nil
}

func parseRepairClock(value string) (time.Time, error) {
	return time.Parse("15:04", value)
}

func adminMonthRange(r *http.Request) (time.Time, time.Time, error) {
	month := r.URL.Query().Get("month")
	if month == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
			time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC),
			nil
	}

	parsed, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	from := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(parsed.Year(), parsed.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	return from, to, nil
}
