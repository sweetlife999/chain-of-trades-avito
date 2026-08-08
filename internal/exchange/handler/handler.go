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

const maxRequestBodyBytes = 1 << 20

type Service interface {
	ListForUser(context.Context, uuid.UUID) ([]exchangemodel.Details, error)
	GetForUser(context.Context, uuid.UUID, uuid.UUID) (exchangemodel.Details, error)
	ConfirmParticipation(context.Context, uuid.UUID, uuid.UUID) error
	DeclineParticipation(context.Context, uuid.UUID, uuid.UUID) error
	CancelByAdmin(context.Context, uuid.UUID) error
	CompleteParticipation(context.Context, uuid.UUID, uuid.UUID) error
	PostMessage(context.Context, uuid.UUID, uuid.UUID, string) (exchangemodel.Message, error)
	ListMessages(context.Context, uuid.UUID, uuid.UUID) ([]exchangemodel.Message, error)
	MarkThreadRead(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
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
	router.With(requireAuth).Post("/exchanges/{id}/complete", h.complete)
	router.With(requireAuth).Post("/exchanges/{id}/messages", h.postMessage)
	router.With(requireAuth).Get("/exchanges/{id}/messages", h.listMessages)
	router.With(requireAuth).Post("/exchanges/{id}/messages/read", h.markRead)
}

// RegisterAdminRoutes вызывается только внутри общей /admin-группы, где уже стоят
// middleware аутентификации и проверки роли администратора.
func (h *Handler) RegisterAdminRoutes(router chi.Router) {
	router.Post("/exchanges/{id}/cancel", h.cancelByAdmin)
}

// @Summary     Принудительно отменить обмен
// @Description Закрывает proposed или confirmed обмен. Для confirmed освобождает объявления и запускает повторный поиск. Решения участников и их статистика не изменяются.
// @Tags        admin
// @Param       id path string true "UUID обмена"
// @Success     204 "Обмен отменён"
// @Failure     400 {object} exchangedto.ErrorResponse "ID не является UUID"
// @Failure     401 {object} exchangedto.ErrorResponse "Пользователь не авторизован"
// @Failure     403 {object} exchangedto.ErrorResponse "Недостаточно прав"
// @Failure     404 {object} exchangedto.ErrorResponse "Обмен не найден"
// @Failure     409 {object} exchangedto.ErrorResponse "Обмен уже завершён или отменён либо объявления находятся в несовместимом состоянии"
// @Failure     500 {object} exchangedto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/exchanges/{id}/cancel [post]
func (h *Handler) cancelByAdmin(w http.ResponseWriter, r *http.Request) {
	exchangeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid exchange id")
		return
	}

	if err := h.service.CancelByAdmin(r.Context(), exchangeID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary     Получить свои обмены
// @Description Возвращает все найденные обмены текущего пользователя вместе с участниками и объявлениями.
// @Tags        exchanges
// @Produce     json
// @Success     200 {array}  exchangedto.ExchangeResponse "Список обменов; если обменов нет, возвращается []"
// @Failure     401 {object} exchangedto.ErrorResponse    "Пользователь не авторизован"
// @Failure     500 {object} exchangedto.ErrorResponse    "Внутренняя ошибка"
// @Security    CookieAuth
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
// @Security    CookieAuth
// @Router      /exchanges/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	exchangeID, userID, ok := exchangeRequest(w, r)
	if !ok {
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
// @Security    CookieAuth
// @Router      /exchanges/{id}/confirm [post]
func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	h.handleDecision(w, r, h.service.ConfirmParticipation)
}

// @Summary     Отказаться от участия в обмене
// @Description Отменяет предложенный или подтверждённый обмен. Для подтверждённого обмена освобождает объявления, увеличивает счётчик сорванных обменов отказавшегося пользователя и запускает поиск новых вариантов.
// @Tags        exchanges
// @Param       id path string true "UUID обмена"
// @Success     204 "Обмен отменён, доступные объявления возвращены в поиск"
// @Failure     400 {object} exchangedto.ErrorResponse "ID не является UUID"
// @Failure     401 {object} exchangedto.ErrorResponse "Пользователь не авторизован"
// @Failure     403 {object} exchangedto.ErrorResponse "Пользователь не участвует в обмене"
// @Failure     404 {object} exchangedto.ErrorResponse "Обмен не найден"
// @Failure     409 {object} exchangedto.ErrorResponse "Обмен уже закрыт, объявление недоступно или получение уже подтверждено"
// @Failure     500 {object} exchangedto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /exchanges/{id}/decline [post]
func (h *Handler) decline(w http.ResponseWriter, r *http.Request) {
	h.handleDecision(w, r, h.service.DeclineParticipation)
}

// @Summary     Подтвердить получение вещи
// @Description Сохраняет подтверждение текущего участника. После подтверждения всеми участниками обмен завершается, объявления помечаются переданными, а счётчики завершённых обменов увеличиваются.
// @Tags        exchanges
// @Param       id path string true "UUID обмена"
// @Success     204 "Получение подтверждено"
// @Failure     400 {object} exchangedto.ErrorResponse "ID не является UUID"
// @Failure     401 {object} exchangedto.ErrorResponse "Пользователь не авторизован"
// @Failure     403 {object} exchangedto.ErrorResponse "Пользователь не участвует в обмене"
// @Failure     404 {object} exchangedto.ErrorResponse "Обмен не найден"
// @Failure     409 {object} exchangedto.ErrorResponse "Обмен ещё не подтверждён или уже отменён"
// @Failure     500 {object} exchangedto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /exchanges/{id}/complete [post]
func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	h.handleDecision(w, r, h.service.CompleteParticipation)
}

// @Summary     Написать в тред обмена
// @Description Добавляет сообщение участника в обсуждение обмена. Писать могут только участники и только пока обмен не закрыт.
// @Tags        exchanges
// @Accept      json
// @Produce     json
// @Param       id      path string                       true "UUID обмена"
// @Param       message body exchangedto.CreateMessageRequest true "Текст сообщения"
// @Success     201 {object} exchangedto.MessageResponse "Сообщение добавлено"
// @Failure     400 {object} exchangedto.ErrorResponse   "ID не является UUID или текст пустой либо длиннее 2000 символов"
// @Failure     401 {object} exchangedto.ErrorResponse   "Пользователь не авторизован"
// @Failure     403 {object} exchangedto.ErrorResponse   "Пользователь не участвует в обмене"
// @Failure     404 {object} exchangedto.ErrorResponse   "Обмен не найден"
// @Failure     409 {object} exchangedto.ErrorResponse   "Обмен закрыт: тред доступен только на чтение"
// @Failure     500 {object} exchangedto.ErrorResponse   "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /exchanges/{id}/messages [post]
func (h *Handler) postMessage(w http.ResponseWriter, r *http.Request) {
	exchangeID, userID, ok := exchangeRequest(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var request exchangedto.CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	message, err := h.service.PostMessage(r.Context(), exchangeID, userID, request.Body)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, exchangedto.MessageFromModel(message))
}

// @Summary     Прочитать тред обмена
// @Description Возвращает обсуждение и события обмена одной лентой в хронологическом порядке. Доступен только участнику.
// @Tags        exchanges
// @Produce     json
// @Param       id path string true "UUID обмена"
// @Success     200 {array}  exchangedto.MessageResponse "Тред; если сообщений нет, возвращается []"
// @Failure     400 {object} exchangedto.ErrorResponse   "ID не является UUID"
// @Failure     401 {object} exchangedto.ErrorResponse   "Пользователь не авторизован"
// @Failure     403 {object} exchangedto.ErrorResponse   "Пользователь не участвует в обмене"
// @Failure     404 {object} exchangedto.ErrorResponse   "Обмен не найден"
// @Failure     500 {object} exchangedto.ErrorResponse   "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /exchanges/{id}/messages [get]
func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	exchangeID, userID, ok := exchangeRequest(w, r)
	if !ok {
		return
	}

	messages, err := h.service.ListMessages(r.Context(), exchangeID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, exchangedto.MessagesFromModels(messages))
}

// @Summary     Отметить тред прочитанным
// @Description Двигает отметку участника до указанного сообщения, обнуляя unread_count в
// @Description GET /exchanges. Клиент присылает id последнего сообщения, которое реально показал,
// @Description поэтому сообщение, пришедшее позже, прочитанным не считается.
// @Description Вызов идемпотентен и назад отметку не двигает: повтор, вторая вкладка и id из
// @Description чужого треда ничего не меняют.
// @Tags        exchanges
// @Accept      json
// @Param       id   path string                        true "UUID обмена"
// @Param       read body exchangedto.MarkReadRequest true "ID последнего показанного сообщения"
// @Success     204 "Отметка обновлена"
// @Failure     400 {object} exchangedto.ErrorResponse "ID не является UUID или тело не JSON"
// @Failure     401 {object} exchangedto.ErrorResponse "Пользователь не авторизован"
// @Failure     403 {object} exchangedto.ErrorResponse "Пользователь не участвует в обмене"
// @Failure     404 {object} exchangedto.ErrorResponse "Обмен не найден"
// @Failure     500 {object} exchangedto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /exchanges/{id}/messages/read [post]
func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	exchangeID, userID, ok := exchangeRequest(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var request exchangedto.MarkReadRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	lastMessageID, err := uuid.Parse(request.LastMessageID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid last message id")
		return
	}

	if err := h.service.MarkThreadRead(r.Context(), exchangeID, userID, lastMessageID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDecision(
	w http.ResponseWriter,
	r *http.Request,
	decision func(context.Context, uuid.UUID, uuid.UUID) error,
) {
	exchangeID, userID, ok := exchangeRequest(w, r)
	if !ok {
		return
	}

	if err := decision(r.Context(), exchangeID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// exchangeRequest достаёт то, что нужно каждому маршруту с {id}: кто спрашивает и о каком
// обмене. Ответ клиенту уже отправлен, если вернулось false.
func exchangeRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}

	exchangeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid exchange id")
		return uuid.Nil, uuid.Nil, false
	}

	return exchangeID, userID, true
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
	case errors.Is(err, exchangeservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
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
