package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	notificationdto "github.com/sweetlife999/chain-of-trades-avito/internal/notification/dto"
	notificationmodel "github.com/sweetlife999/chain-of-trades-avito/internal/notification/model"
	notificationservice "github.com/sweetlife999/chain-of-trades-avito/internal/notification/service"
)

const defaultPageLimit int32 = 50

type Service interface {
	List(context.Context, uuid.UUID, notificationmodel.Filter) (notificationmodel.Page, error)
	MarkRead(context.Context, uuid.UUID, uuid.UUID) error
	MarkAllRead(context.Context, uuid.UUID) (int64, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	router.With(requireAuth).Get("/notifications", h.list)
	router.With(requireAuth).Post("/notifications/read-all", h.markAllRead)
	router.With(requireAuth).Post("/notifications/{id}/read", h.markRead)
}

// @Summary     Получить уведомления
// @Description Возвращает уведомления текущего пользователя, число всех непрочитанных и контекст обмена.
// @Tags        notifications
// @Produce     json
// @Param       unread_only query bool false "Вернуть только непрочитанные" default(false)
// @Param       limit       query int  false "Размер страницы, от 1 до 100" default(50)
// @Param       offset      query int  false "Смещение, начиная с 0" default(0)
// @Success     200 {object} notificationdto.PageResponse
// @Failure     400 {object} notificationdto.ErrorResponse "Некорректные параметры"
// @Failure     401 {object} notificationdto.ErrorResponse "Пользователь не авторизован"
// @Failure     500 {object} notificationdto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /notifications [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	filter, err := parseFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := h.service.List(r.Context(), userID, filter)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, notificationdto.FromPage(page))
}

// @Summary     Отметить уведомление прочитанным
// @Tags        notifications
// @Param       id path string true "UUID уведомления"
// @Success     204
// @Failure     400 {object} notificationdto.ErrorResponse "Некорректный UUID"
// @Failure     401 {object} notificationdto.ErrorResponse "Пользователь не авторизован"
// @Failure     404 {object} notificationdto.ErrorResponse "Уведомление не найдено у пользователя"
// @Failure     500 {object} notificationdto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /notifications/{id}/read [post]
func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	notificationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := h.service.MarkRead(r.Context(), userID, notificationID); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary     Отметить все уведомления прочитанными
// @Tags        notifications
// @Produce     json
// @Success     200 {object} notificationdto.MarkAllResponse
// @Failure     401 {object} notificationdto.ErrorResponse "Пользователь не авторизован"
// @Failure     500 {object} notificationdto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /notifications/read-all [post]
func (h *Handler) markAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	affected, err := h.service.MarkAllRead(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, notificationdto.MarkAllResponse{MarkedCount: affected})
}

func parseFilter(r *http.Request) (notificationmodel.Filter, error) {
	filter := notificationmodel.Filter{Limit: defaultPageLimit}
	query := r.URL.Query()

	if raw := query.Get("unread_only"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return notificationmodel.Filter{}, errors.New("unread_only must be true or false")
		}
		filter.UnreadOnly = value
	}
	if raw := query.Get("limit"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return notificationmodel.Filter{}, errors.New("limit must be an integer")
		}
		filter.Limit = int32(value)
	}
	if raw := query.Get("offset"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return notificationmodel.Filter{}, errors.New("offset must be an integer")
		}
		filter.Offset = int32(value)
	}
	return filter, nil
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, notificationservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, notificationservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "notification not found")
	default:
		log.Printf("notification handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, notificationdto.ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode notification response: %v", err)
	}
}
