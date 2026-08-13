package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	supportdto "github.com/sweetlife999/chain-of-trades-avito/internal/support/dto"
	supportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/support/model"
)

type AdminService interface {
	List(context.Context, supportmodel.AdminFilter) (supportmodel.AdminPage, error)
	Messages(context.Context, uuid.UUID) (supportmodel.Thread, []supportmodel.Message, error)
	Assign(context.Context, uuid.UUID, uuid.UUID) (supportmodel.Thread, error)
	Send(context.Context, uuid.UUID, uuid.UUID, string) (supportmodel.Message, error)
	Close(context.Context, uuid.UUID, uuid.UUID) (supportmodel.Thread, error)
}

type AdminHandler struct{ service AdminService }

func NewAdmin(service AdminService) *AdminHandler { return &AdminHandler{service: service} }

func (h *AdminHandler) RegisterRoutes(router chi.Router) {
	router.Get("/support/threads", h.list)
	router.Get("/support/threads/{id}/messages", h.messages)
	router.Post("/support/threads/{id}/assign", h.assign)
	router.Post("/support/threads/{id}/messages", h.send)
	router.Post("/support/threads/{id}/close", h.close)
}

// @Summary     Очередь обращений в поддержку
// @Description Сначала открытые, затем обращения в работе и закрытые. Поддерживает фильтр и пагинацию.
// @Tags        admin-support
// @Produce     json
// @Param       status query string false "Статус" Enums(open,in_progress,closed)
// @Param       needs_human query bool false "Только ждущие модератора: эскалированные и те, на которые никто не ответил" default(false)
// @Param       limit query int false "Размер страницы (1-100)" default(20)
// @Param       offset query int false "Смещение" default(0)
// @Success     200 {object} supportdto.AdminPageResponse
// @Failure     400 {object} supportdto.ErrorResponse
// @Failure     401 {object} supportdto.ErrorResponse
// @Failure     403 {object} supportdto.ErrorResponse
// @Failure     500 {object} supportdto.ErrorResponse
// @Security    CookieAuth
// @Router      /admin/support/threads [get]
func (h *AdminHandler) list(w http.ResponseWriter, r *http.Request) {
	filter, err := adminFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := h.service.List(r.Context(), filter)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, supportdto.PageFromModel(page))
}

// @Summary  Переписка обращения для администратора
// @Tags     admin-support
// @Produce  json
// @Param    id path string true "UUID обращения"
// @Success  200 {object} supportdto.ThreadMessagesResponse
// @Failure  400 {object} supportdto.ErrorResponse
// @Failure  401 {object} supportdto.ErrorResponse
// @Failure  403 {object} supportdto.ErrorResponse
// @Failure  404 {object} supportdto.ErrorResponse
// @Failure  500 {object} supportdto.ErrorResponse
// @Security CookieAuth
// @Router   /admin/support/threads/{id}/messages [get]
func (h *AdminHandler) messages(w http.ResponseWriter, r *http.Request) {
	threadID, ok := requestThreadID(w, r)
	if !ok {
		return
	}
	thread, messages, err := h.service.Messages(r.Context(), threadID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, supportdto.MessagesResponse(thread, messages))
}

// @Summary     Взять обращение в работу
// @Description Назначает открытое обращение текущему администратору.
// @Tags        admin-support
// @Produce     json
// @Param       id path string true "UUID обращения"
// @Success     200 {object} supportdto.ThreadResponse
// @Failure     400 {object} supportdto.ErrorResponse
// @Failure     401 {object} supportdto.ErrorResponse
// @Failure     403 {object} supportdto.ErrorResponse
// @Failure     404 {object} supportdto.ErrorResponse
// @Failure     409 {object} supportdto.ErrorResponse
// @Failure     500 {object} supportdto.ErrorResponse
// @Security    CookieAuth
// @Router      /admin/support/threads/{id}/assign [post]
func (h *AdminHandler) assign(w http.ResponseWriter, r *http.Request) {
	threadID, ok := requestThreadID(w, r)
	if !ok {
		return
	}
	adminID, ok := requestAdminID(w, r)
	if !ok {
		return
	}
	thread, err := h.service.Assign(r.Context(), threadID, adminID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, supportdto.ThreadFromModel(thread))
}

// @Summary  Ответить пользователю от имени поддержки
// @Tags     admin-support
// @Accept   json
// @Produce  json
// @Param    id path string true "UUID обращения"
// @Param    request body supportdto.CreateMessageRequest true "Текст сообщения"
// @Success  201 {object} supportdto.MessageResponse
// @Failure  400 {object} supportdto.ErrorResponse
// @Failure  401 {object} supportdto.ErrorResponse
// @Failure  403 {object} supportdto.ErrorResponse
// @Failure  404 {object} supportdto.ErrorResponse
// @Failure  409 {object} supportdto.ErrorResponse "Обращение назначено другому админу или закрыто"
// @Failure  500 {object} supportdto.ErrorResponse
// @Security CookieAuth
// @Router   /admin/support/threads/{id}/messages [post]
func (h *AdminHandler) send(w http.ResponseWriter, r *http.Request) {
	threadID, ok := requestThreadID(w, r)
	if !ok {
		return
	}
	adminID, ok := requestAdminID(w, r)
	if !ok {
		return
	}
	var request supportdto.CreateMessageRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	message, err := h.service.Send(r.Context(), threadID, adminID, request.Body)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, supportdto.MessageFromModel(message))
}

// @Summary  Закрыть обращение администратором
// @Tags     admin-support
// @Produce  json
// @Param    id path string true "UUID обращения"
// @Success  200 {object} supportdto.ThreadResponse
// @Failure  400 {object} supportdto.ErrorResponse
// @Failure  401 {object} supportdto.ErrorResponse
// @Failure  403 {object} supportdto.ErrorResponse
// @Failure  404 {object} supportdto.ErrorResponse
// @Failure  409 {object} supportdto.ErrorResponse
// @Failure  500 {object} supportdto.ErrorResponse
// @Security CookieAuth
// @Router   /admin/support/threads/{id}/close [post]
func (h *AdminHandler) close(w http.ResponseWriter, r *http.Request) {
	threadID, ok := requestThreadID(w, r)
	if !ok {
		return
	}
	adminID, ok := requestAdminID(w, r)
	if !ok {
		return
	}
	thread, err := h.service.Close(r.Context(), threadID, adminID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, supportdto.ThreadFromModel(thread))
}

func requestAdminID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	adminID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
	}
	return adminID, ok
}

func adminFilter(r *http.Request) (supportmodel.AdminFilter, error) {
	filter := supportmodel.AdminFilter{Status: r.URL.Query().Get("status")}
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		filter.Limit, err = parseInt32(raw)
		if err != nil {
			return filter, err
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		filter.Offset, err = parseInt32(raw)
		if err != nil {
			return filter, err
		}
	}
	// Умолчание — показать всё: очередь без параметров обязана оставаться полной, а
	// сужает её тот, кто явно попросил. Отсеивать по умолчанию на уровне API значило бы
	// прятать обращения от любого клиента, который про этот параметр не знает.
	if raw := r.URL.Query().Get("needs_human"); raw != "" {
		filter.NeedsHuman, err = strconv.ParseBool(raw)
		if err != nil {
			return filter, err
		}
	}
	return filter, nil
}

func parseInt32(value string) (int32, error) {
	parsed, err := strconv.ParseInt(value, 10, 32)
	return int32(parsed), err
}
