package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	ratingdto "github.com/sweetlife999/chain-of-trades-avito/internal/rating/dto"
	ratingmodel "github.com/sweetlife999/chain-of-trades-avito/internal/rating/model"
	ratingservice "github.com/sweetlife999/chain-of-trades-avito/internal/rating/service"
)

const maxRequestBodyBytes = 1 << 20

type Service interface {
	Rate(context.Context, uuid.UUID, uuid.UUID, int32, string) (ratingmodel.Rating, error)
	ListForUser(context.Context, uuid.UUID, int32, int32) (ratingmodel.Page, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	// Имя параметра совпадает с уже занятым на этой позиции у соседних маршрутов:
	// chi не разрешает два разных имени в одном сегменте и падает при старте.
	router.With(requireAuth).Put("/exchanges/{id}/rating", h.rate)
	// Лента публична, как и сам профиль: рейтинг нужен до того, как решишься меняться.
	router.Get("/users/{id}/ratings", h.list)
}

// @Summary     Оценить партнёра по обмену
// @Description Ставит оценку участнику, который передал вещь текущему пользователю. Кого
// @Description оценивают, определяет сама цепочка. Повторный запрос переписывает оценку.
// @Description Доступно 14 дней после завершения обмена.
// @Tags        ratings
// @Accept      json
// @Produce     json
// @Param       id      path string                true "UUID обмена"
// @Param       request body ratingdto.RateRequest true "Балл от 1 до 5 и необязательный комментарий"
// @Success     200 {object} ratingdto.RatingResponse
// @Failure     400 {object} ratingdto.RatingError "Некорректный UUID, тело или балл вне 1..5"
// @Failure     401 {object} ratingdto.RatingError "Пользователь не авторизован"
// @Failure     403 {object} ratingdto.RatingError "Пользователь не участвует в обмене"
// @Failure     404 {object} ratingdto.RatingError "Обмен не найден"
// @Failure     409 {object} ratingdto.RatingError "Обмен не завершён или срок оценки истёк"
// @Failure     500 {object} ratingdto.RatingError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /exchanges/{id}/rating [put]
func (h *Handler) rate(w http.ResponseWriter, r *http.Request) {
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	exchangeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid exchange id")
		return
	}

	var request ratingdto.RateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rating, err := h.service.Rate(r.Context(), exchangeID, userID, request.Score, request.Comment)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ratingdto.FromModel(rating))
}

// @Summary     Получить отзывы о пользователе
// @Description Возвращает оценки, полученные пользователем, от новых к старым. Отзывы
// @Description анонимны: автор не возвращается. Общее число оценок лежит в профиле.
// @Tags        ratings
// @Produce     json
// @Param       id     path  string true  "UUID пользователя"
// @Param       limit  query int    false "Размер страницы, от 1 до 100" default(20)
// @Param       offset query int    false "Смещение, начиная с 0" default(0)
// @Success     200 {object} ratingdto.RatingsPageResponse
// @Failure     400 {object} ratingdto.RatingError "Некорректный UUID или параметры страницы"
// @Failure     500 {object} ratingdto.RatingError "Внутренняя ошибка"
// @Router      /users/{id}/ratings [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	limit, err := int32Query(r, "limit", ratingservice.DefaultLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, "limit must be an integer")
		return
	}
	offset, err := int32Query(r, "offset", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, "offset must be an integer")
		return
	}

	page, err := h.service.ListForUser(r.Context(), userID, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ratingdto.FromPage(page))
}

func int32Query(r *http.Request, name string, fallback int32) (int32, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(value), nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ratingservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ratingservice.ErrForbidden):
		writeError(w, http.StatusForbidden, "not an exchange participant")
	case errors.Is(err, ratingservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "exchange not found")
	case errors.Is(err, ratingservice.ErrNotCompleted):
		writeError(w, http.StatusConflict, "exchange is not completed")
	case errors.Is(err, ratingservice.ErrWindowClosed):
		writeError(w, http.StatusConflict, "rating window has closed")
	default:
		log.Printf("rating handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ratingdto.RatingError{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode rating response: %v", err)
	}
}
