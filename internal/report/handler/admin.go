package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	reportdto "github.com/sweetlife999/chain-of-trades-avito/internal/report/dto"
	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
	reportservice "github.com/sweetlife999/chain-of-trades-avito/internal/report/service"
)

type AdminService interface {
	List(context.Context, reportmodel.AdminFilter) (reportmodel.AdminPage, error)
	Get(context.Context, uuid.UUID) (reportmodel.AdminReport, error)
	Assign(context.Context, uuid.UUID, uuid.UUID) (reportmodel.AdminReport, error)
	Decide(context.Context, uuid.UUID, uuid.UUID, string, string) (reportmodel.AdminReport, error)
	ListMessages(
		context.Context,
		uuid.UUID,
	) (reportmodel.AdminReport, []exchangemodel.Message, error)
}

type AdminHandler struct {
	service AdminService
}

func NewAdmin(service AdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) RegisterRoutes(router chi.Router) {
	router.Get("/reports", h.list)
	router.Get("/reports/{id}", h.get)
	router.Get("/reports/{id}/messages", h.listMessages)
	router.Post("/reports/{id}/assign", h.assign)
	router.Post("/reports/{id}/resolve", h.resolve)
	router.Post("/reports/{id}/reject", h.reject)
}

// @Summary     Очередь жалоб
// @Description Возвращает жалобы для модерации вместе с жалобщиком, нарушителем, сообщением и обменом.
// @Description Поддерживает пагинацию и фильтры по статусу, причине и назначенному администратору.
// @Tags        admin-reports
// @Produce     json
// @Param       status query string false "Статус жалобы" Enums(open,resolved,rejected)
// @Param       reason query string false "Причина жалобы" Enums(spam,abuse,other)
// @Param       assignee_id query string false "UUID назначенного администратора"
// @Param       limit query int false "Размер страницы (1-100)" default(20)
// @Param       offset query int false "Смещение" default(0)
// @Success     200 {object} reportdto.AdminReportListResponse
// @Failure     400 {object} reportdto.ReportError "Некорректный фильтр или пагинация"
// @Failure     401 {object} reportdto.ReportError "Пользователь не авторизован"
// @Failure     403 {object} reportdto.ReportError "Пользователь не администратор"
// @Failure     500 {object} reportdto.ReportError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/reports [get]
func (h *AdminHandler) list(w http.ResponseWriter, r *http.Request) {
	filter, err := adminFilterFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	page, err := h.service.List(r.Context(), filter)
	if err != nil {
		handleAdminServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reportdto.AdminPageFromModel(page))
}

// @Summary     Карточка жалобы
// @Description Возвращает жалобу, автора жалобы, автора сообщения, само сообщение и обмен.
// @Tags        admin-reports
// @Produce     json
// @Param       id path string true "UUID жалобы"
// @Success     200 {object} reportdto.AdminReportResponse
// @Failure     400 {object} reportdto.ReportError "Некорректный UUID"
// @Failure     401 {object} reportdto.ReportError "Пользователь не авторизован"
// @Failure     403 {object} reportdto.ReportError "Пользователь не администратор"
// @Failure     404 {object} reportdto.ReportError "Жалоба не найдена"
// @Failure     500 {object} reportdto.ReportError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/reports/{id} [get]
func (h *AdminHandler) get(w http.ResponseWriter, r *http.Request) {
	reportID, ok := reportIDFromRequest(w, r)
	if !ok {
		return
	}

	report, err := h.service.Get(r.Context(), reportID)
	if err != nil {
		handleAdminServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reportdto.AdminReportFromModel(report))
}

// @Summary     Переписка по жалобе
// @Description Возвращает полный тред обмена, к которому относится жалоба. Доступ только на чтение.
// @Tags        admin-reports
// @Produce     json
// @Param       id path string true "UUID жалобы"
// @Success     200 {object} reportdto.AdminReportMessagesResponse
// @Failure     400 {object} reportdto.ReportError "Некорректный UUID"
// @Failure     401 {object} reportdto.ReportError "Пользователь не авторизован"
// @Failure     403 {object} reportdto.ReportError "Пользователь не администратор"
// @Failure     404 {object} reportdto.ReportError "Жалоба не найдена"
// @Failure     500 {object} reportdto.ReportError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/reports/{id}/messages [get]
func (h *AdminHandler) listMessages(w http.ResponseWriter, r *http.Request) {
	reportID, ok := reportIDFromRequest(w, r)
	if !ok {
		return
	}

	report, messages, err := h.service.ListMessages(r.Context(), reportID)
	if err != nil {
		handleAdminServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reportdto.AdminMessagesFromModels(report, messages))
}

// @Summary     Взять жалобу в работу
// @Description Атомарно назначает открытую жалобу текущему администратору.
// @Description Если её уже взял другой администратор или она закрыта, возвращает конфликт.
// @Tags        admin-reports
// @Produce     json
// @Param       id path string true "UUID жалобы"
// @Success     200 {object} reportdto.AdminReportResponse
// @Failure     400 {object} reportdto.ReportError "Некорректный UUID"
// @Failure     401 {object} reportdto.ReportError "Пользователь не авторизован"
// @Failure     403 {object} reportdto.ReportError "Пользователь не администратор"
// @Failure     404 {object} reportdto.ReportError "Жалоба не найдена"
// @Failure     409 {object} reportdto.ReportError "Жалоба уже назначена или закрыта"
// @Failure     500 {object} reportdto.ReportError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/reports/{id}/assign [post]
func (h *AdminHandler) assign(w http.ResponseWriter, r *http.Request) {
	reportID, ok := reportIDFromRequest(w, r)
	if !ok {
		return
	}
	adminID, ok := adminIDFromRequest(w, r)
	if !ok {
		return
	}

	report, err := h.service.Assign(r.Context(), reportID, adminID)
	if err != nil {
		handleAdminServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reportdto.AdminReportFromModel(report))
}

// @Summary     Подтвердить жалобу
// @Description Закрывает назначенную текущему администратору жалобу со статусом resolved.
// @Tags        admin-reports
// @Accept      json
// @Produce     json
// @Param       id path string true "UUID жалобы"
// @Param       request body reportdto.AdminDecisionRequest true "Комментарий к решению"
// @Success     200 {object} reportdto.AdminReportResponse
// @Failure     400 {object} reportdto.ReportError "Некорректный запрос"
// @Failure     401 {object} reportdto.ReportError "Пользователь не авторизован"
// @Failure     403 {object} reportdto.ReportError "Пользователь не администратор"
// @Failure     404 {object} reportdto.ReportError "Жалоба не найдена"
// @Failure     409 {object} reportdto.ReportError "Жалоба не назначена этому админу или уже закрыта"
// @Failure     500 {object} reportdto.ReportError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/reports/{id}/resolve [post]
func (h *AdminHandler) resolve(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "resolved")
}

// @Summary     Отклонить жалобу
// @Description Закрывает назначенную текущему администратору жалобу со статусом rejected.
// @Tags        admin-reports
// @Accept      json
// @Produce     json
// @Param       id path string true "UUID жалобы"
// @Param       request body reportdto.AdminDecisionRequest true "Комментарий к решению"
// @Success     200 {object} reportdto.AdminReportResponse
// @Failure     400 {object} reportdto.ReportError "Некорректный запрос"
// @Failure     401 {object} reportdto.ReportError "Пользователь не авторизован"
// @Failure     403 {object} reportdto.ReportError "Пользователь не администратор"
// @Failure     404 {object} reportdto.ReportError "Жалоба не найдена"
// @Failure     409 {object} reportdto.ReportError "Жалоба не назначена этому админу или уже закрыта"
// @Failure     500 {object} reportdto.ReportError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/reports/{id}/reject [post]
func (h *AdminHandler) reject(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "rejected")
}

func (h *AdminHandler) decide(w http.ResponseWriter, r *http.Request, decision string) {
	reportID, ok := reportIDFromRequest(w, r)
	if !ok {
		return
	}
	adminID, ok := adminIDFromRequest(w, r)
	if !ok {
		return
	}

	var request reportdto.AdminDecisionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	report, err := h.service.Decide(
		r.Context(),
		reportID,
		adminID,
		decision,
		request.Comment,
	)
	if err != nil {
		handleAdminServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reportdto.AdminReportFromModel(report))
}

func adminFilterFromRequest(r *http.Request) (reportmodel.AdminFilter, error) {
	query := r.URL.Query()
	filter := reportmodel.AdminFilter{
		Status: query.Get("status"),
		Reason: query.Get("reason"),
	}

	if value := query.Get("assignee_id"); value != "" {
		assigneeID, err := uuid.Parse(value)
		if err != nil {
			return reportmodel.AdminFilter{}, errors.New("invalid assignee_id")
		}
		filter.AssigneeID = &assigneeID
	}

	limit, err := int32Query(query.Get("limit"), reportservice.DefaultAdminLimit)
	if err != nil {
		return reportmodel.AdminFilter{}, errors.New("invalid limit")
	}
	offset, err := int32Query(query.Get("offset"), 0)
	if err != nil {
		return reportmodel.AdminFilter{}, errors.New("invalid offset")
	}
	filter.Limit = limit
	filter.Offset = offset

	return filter, nil
}

func int32Query(value string, defaultValue int32) (int32, error) {
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(parsed), nil
}

func reportIDFromRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	reportID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid report id")
		return uuid.Nil, false
	}

	return reportID, true
}

func adminIDFromRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	adminID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}

	return adminID, true
}

func handleAdminServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, reportservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, reportservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "report not found")
	case errors.Is(err, reportservice.ErrAlreadyAssigned):
		writeError(w, http.StatusConflict, "report is already assigned")
	case errors.Is(err, reportservice.ErrNotAssigned):
		writeError(w, http.StatusConflict, "report must be assigned before processing")
	case errors.Is(err, reportservice.ErrAssignedToOther):
		writeError(w, http.StatusConflict, "report is assigned to another administrator")
	case errors.Is(err, reportservice.ErrAlreadyProcessed):
		writeError(w, http.StatusConflict, "report is already processed")
	default:
		log.Printf("admin report handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
