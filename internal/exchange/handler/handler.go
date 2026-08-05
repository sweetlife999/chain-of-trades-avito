package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	exchangedto "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/dto"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
)

type Service interface {
	ListForUser(context.Context, uuid.UUID) ([]exchangemodel.Details, error)
	GetForUser(context.Context, uuid.UUID, uuid.UUID) (exchangemodel.Details, error)
	ConfirmParticipation(context.Context, uuid.UUID, uuid.UUID) error
	DeclineParticipation(context.Context, uuid.UUID, uuid.UUID) error
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	router.With(requireAuth).Get("/exchanges", h.list)
	router.With(requireAuth).Get("/exchanges/{id}", h.getByID)
	router.With(requireAuth).Post("/exchanges/{id}/confirm", h.confirm)
	router.With(requireAuth).Post("/exchanges/{id}/decline", h.decline)
}

// @Summary     Получить свои обмены
// @Description Возвращает все найденные обмены текущего пользователя вместе с участниками и объявлениями.
// @Tags        exchanges
// @Produce     json
// @Success     200 {array}  exchangedto.ExchangeResponse "Список обменов; если обменов нет, возвращается []"
// @Failure     401 {object} exchangedto.ErrorResponse    "Пользователь не авторизован"
// @Failure     500 {object} exchangedto.ErrorResponse    "Внутренняя ошибка"
// @Router      /exchanges [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	exchanges, err := h.service.ListForUser(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, exchangedto.FromModels(exchanges))
}

// @Summary     Получить обмен по ID
// @Description Доступен только участнику этого обмена.
// @Tags        exchanges
// @Produce     json
// @Param       id path string true "UUID обмена" example(8db9f3e2-8a45-4a70-b3d1-167b4f97e121)
// @Success     200 {object} exchangedto.ExchangeResponse "Обмен"
// @Failure     400 {object} exchangedto.ErrorResponse    "ID не является UUID"
// @Failure     401 {object} exchangedto.ErrorResponse    "Пользователь не авторизован"
// @Failure     403 {object} exchangedto.ErrorResponse    "Пользователь не участвует в обмене"
// @Failure     404 {object} exchangedto.ErrorResponse    "Обмен не найден"
// @Failure     500 {object} exchangedto.ErrorResponse    "Внутренняя ошибка"
// @Router      /exchanges/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	exchangeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid exchange id")
		return
	}

	exchange, err := h.service.GetForUser(r.Context(), exchangeID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, exchangedto.FromModel(exchange))
}

// @Summary     Подтвердить участие в обмене
// @Description Сохраняет согласие текущего участника. После согласия всех участников обмен подтверждается, а объявления резервируются.
// @Tags        exchanges
// @Param       id path string true "UUID обмена"
// @Success     204 "Участие подтверждено"
// @Failure     400 {object} exchangedto.ErrorResponse "ID не является UUID"
// @Failure     401 {object} exchangedto.ErrorResponse "Пользователь не авторизован"
// @Failure     403 {object} exchangedto.ErrorResponse "Пользователь не участвует в обмене"
// @Failure     404 {object} exchangedto.ErrorResponse "Обмен не найден"
// @Failure     409 {object} exchangedto.ErrorResponse "Решение уже принято, обмен закрыт или объявление недоступно"
// @Failure     500 {object} exchangedto.ErrorResponse "Внутренняя ошибка"
// @Router      /exchanges/{id}/confirm [post]
func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	h.handleDecision(w, r, h.service.ConfirmParticipation)
}

// @Summary     Отказаться от участия в обмене
// @Description Сохраняет отказ текущего участника и отменяет предложенный обмен.
// @Tags        exchanges
// @Param       id path string true "UUID обмена"
// @Success     204 "Отказ сохранён, обмен отменён"
// @Failure     400 {object} exchangedto.ErrorResponse "ID не является UUID"
// @Failure     401 {object} exchangedto.ErrorResponse "Пользователь не авторизован"
// @Failure     403 {object} exchangedto.ErrorResponse "Пользователь не участвует в обмене"
// @Failure     404 {object} exchangedto.ErrorResponse "Обмен не найден"
// @Failure     409 {object} exchangedto.ErrorResponse "Решение уже принято или обмен закрыт"
// @Failure     500 {object} exchangedto.ErrorResponse "Внутренняя ошибка"
// @Router      /exchanges/{id}/decline [post]
func (h *Handler) decline(w http.ResponseWriter, r *http.Request) {
	h.handleDecision(w, r, h.service.DeclineParticipation)
}

func (h *Handler) handleDecision(
	w http.ResponseWriter,
	r *http.Request,
	decision func(context.Context, uuid.UUID, uuid.UUID) error,
) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	exchangeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid exchange id")
		return
	}

	if err := decision(r.Context(), exchangeID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func currentUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}

	return userID, true
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, exchangeservice.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, exchangeservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "exchange not found")
	case errors.Is(err, exchangeservice.ErrConflict):
		writeError(w, http.StatusConflict, "exchange state conflict")
	default:
		log.Printf("exchange handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, exchangedto.ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}
