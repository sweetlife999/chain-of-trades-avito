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

	adminexchangedto "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/dto"
	adminexchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/model"
	adminexchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/service"
)

type Service interface {
	ListActiveByUser(context.Context, uuid.UUID, int32, int32) (adminexchangemodel.Page, error)
	ListActive(context.Context, string, int32, int32) (adminexchangemodel.Page, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes вызывается только внутри уже защищённой группы /admin.
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/exchanges", h.listActive)
	router.Get("/users/{user_id}/exchanges", h.listActiveByUser)
}

// listActive godoc
// @Summary     Получить активные обмены по статусу
// @Description Доступно только администратору. Возвращает очередь обменов нужного этапа вместе с ID, участниками и вещами.
// @Tags        admin exchanges
// @Produce     json
// @Param       status query string true  "Статус обмена" Enums(proposed,confirmed,delivering,delivered)
// @Param       limit  query int    false "Размер страницы (1–100)" default(20) minimum(1) maximum(100)
// @Param       offset query int    false "Смещение от начала списка" default(0) minimum(0)
// @Success     200 {object} adminexchangedto.ListResponse "Активные обмены"
// @Failure     400 {object} adminexchangedto.ErrorResponse "Некорректный статус или пагинация"
// @Failure     401 {object} adminexchangedto.ErrorResponse "Пользователь не авторизован"
// @Failure     403 {object} adminexchangedto.ErrorResponse "Недостаточно прав"
// @Failure     500 {object} adminexchangedto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/exchanges [get]
func (h *Handler) listActive(w http.ResponseWriter, r *http.Request) {
	limit, err := queryInt32(r, "limit", adminexchangeservice.DefaultLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return
	}
	offset, err := queryInt32(r, "offset", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offset")
		return
	}

	page, err := h.service.ListActive(
		r.Context(),
		r.URL.Query().Get("status"),
		limit,
		offset,
	)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, adminexchangedto.FromModel(page))
}

// listActiveByUser godoc
// @Summary     Получить активные обмены пользователя
// @Description Доступно только администратору. Возвращает proposed, confirmed, delivering и delivered обмены пользователя вместе с участниками и вещами.
// @Tags        admin exchanges
// @Produce     json
// @Param       user_id path  string true  "UUID пользователя"
// @Param       limit   query int    false "Размер страницы (1–100)" default(20) minimum(1) maximum(100)
// @Param       offset  query int    false "Смещение от начала списка" default(0) minimum(0)
// @Success     200 {object} adminexchangedto.ListResponse "Активные обмены пользователя"
// @Failure     400 {object} adminexchangedto.ErrorResponse "Некорректный UUID или пагинация"
// @Failure     401 {object} adminexchangedto.ErrorResponse "Пользователь не авторизован"
// @Failure     403 {object} adminexchangedto.ErrorResponse "Недостаточно прав"
// @Failure     404 {object} adminexchangedto.ErrorResponse "Пользователь не найден"
// @Failure     500 {object} adminexchangedto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/users/{user_id}/exchanges [get]
func (h *Handler) listActiveByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	limit, err := queryInt32(r, "limit", adminexchangeservice.DefaultLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return
	}
	offset, err := queryInt32(r, "offset", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offset")
		return
	}

	page, err := h.service.ListActiveByUser(r.Context(), userID, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, adminexchangedto.FromModel(page))
}

func queryInt32(r *http.Request, name string, defaultValue int32) (int32, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(value), nil
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, adminexchangeservice.ErrValidation):
		writeError(w, http.StatusBadRequest, "invalid status or pagination")
	case errors.Is(err, adminexchangeservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	default:
		log.Printf("admin exchanges handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, adminexchangedto.ErrorResponse{Error: message})
}
