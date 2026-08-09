package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	pickuppointdto "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/dto"
	pickuppointmodel "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/model"
	pickuppointservice "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/service"
)

const maxRequestBodyBytes = 1 << 20

type Service interface {
	Create(context.Context, pickuppointservice.CreateInput) (pickuppointmodel.PickupPoint, error)
	GetByID(context.Context, uuid.UUID) (pickuppointmodel.PickupPoint, error)
	List(context.Context) ([]pickuppointmodel.PickupPoint, error)
	Update(context.Context, uuid.UUID, pickuppointservice.UpdateInput) (pickuppointmodel.PickupPoint, error)
	Delete(context.Context, uuid.UUID) error
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes вызывается только внутри уже защищённой группы /admin. Чтение здесь
// дублирует публичные маршруты намеренно: на /admin/pickup-points ходит админка фронта.
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/pickup-points", h.create)
	router.Get("/pickup-points", h.list)
	router.Get("/pickup-points/{id}", h.getByID)
	router.Patch("/pickup-points/{id}", h.update)
	router.Delete("/pickup-points/{id}", h.delete)
}

// RegisterPublicRoutes отдаёт справочник пунктов любому авторизованному пользователю:
// без него нечего выбрать в POST /items/{id}/pickup. Запись остаётся за администратором.
func (h *Handler) RegisterPublicRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	router.With(requireAuth).Get("/pickup-points", h.list)
	router.With(requireAuth).Get("/pickup-points/{id}", h.getByID)
}

// @Summary     Создать ПВЗ
// @Description Доступно только администратору.
// @Tags        admin pickup-points
// @Accept      json
// @Produce     json
// @Param       request body pickuppointdto.CreatePickupPointRequest true "Данные ПВЗ"
// @Success     201 {object} pickuppointdto.PickupPointResponse "ПВЗ создан"
// @Failure     400 {object} pickuppointdto.PickupPointError "Некорректные данные"
// @Failure     401 {object} pickuppointdto.PickupPointError "Пользователь не авторизован"
// @Failure     403 {object} pickuppointdto.PickupPointError "Недостаточно прав"
// @Failure     500 {object} pickuppointdto.PickupPointError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/pickup-points [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var request pickuppointdto.CreatePickupPointRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	point, err := h.service.Create(r.Context(), pickuppointservice.CreateInput{
		Name:    request.Name,
		Address: request.Address,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	w.Header().Set("Location", "/admin/pickup-points/"+point.ID.String())
	writeJSON(w, http.StatusCreated, pickuppointdto.FromModel(point))
}

// @Summary     Получить список ПВЗ
// @Description Возвращает ПВЗ от новых к старым. Читать список может любой авторизованный
// @Description пользователь: без него не из чего выбирать пункт для POST /items/{id}/pickup.
// @Description Заводить и править ПВЗ по-прежнему может только администратор.
// @Tags        pickup-points
// @Produce     json
// @Success     200 {array} pickuppointdto.PickupPointResponse "Список ПВЗ"
// @Failure     401 {object} pickuppointdto.PickupPointError "Пользователь не авторизован"
// @Failure     403 {object} pickuppointdto.PickupPointError "Недостаточно прав"
// @Failure     500 {object} pickuppointdto.PickupPointError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /pickup-points [get]
// @Router      /admin/pickup-points [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	points, err := h.service.List(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, pickuppointdto.FromModels(points))
}

// @Summary     Получить ПВЗ по ID
// @Description Доступно любому авторизованному пользователю — по той же причине, что и список.
// @Tags        pickup-points
// @Produce     json
// @Param       id path string true "UUID ПВЗ"
// @Success     200 {object} pickuppointdto.PickupPointResponse "ПВЗ"
// @Failure     400 {object} pickuppointdto.PickupPointError "Некорректный UUID"
// @Failure     401 {object} pickuppointdto.PickupPointError "Пользователь не авторизован"
// @Failure     403 {object} pickuppointdto.PickupPointError "Недостаточно прав"
// @Failure     404 {object} pickuppointdto.PickupPointError "ПВЗ не найден"
// @Failure     500 {object} pickuppointdto.PickupPointError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /pickup-points/{id} [get]
// @Router      /admin/pickup-points/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	point, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, pickuppointdto.FromModel(point))
}

// @Summary     Изменить ПВЗ
// @Description Доступно только администратору. Достаточно передать одно поле.
// @Tags        admin pickup-points
// @Accept      json
// @Produce     json
// @Param       id path string true "UUID ПВЗ"
// @Param       request body pickuppointdto.UpdatePickupPointRequest true "Изменяемые поля"
// @Success     200 {object} pickuppointdto.PickupPointResponse "ПВЗ обновлён"
// @Failure     400 {object} pickuppointdto.PickupPointError "Некорректные данные или UUID"
// @Failure     401 {object} pickuppointdto.PickupPointError "Пользователь не авторизован"
// @Failure     403 {object} pickuppointdto.PickupPointError "Недостаточно прав"
// @Failure     404 {object} pickuppointdto.PickupPointError "ПВЗ не найден"
// @Failure     500 {object} pickuppointdto.PickupPointError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/pickup-points/{id} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var request pickuppointdto.UpdatePickupPointRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	point, err := h.service.Update(r.Context(), id, pickuppointservice.UpdateInput{
		Name:    request.Name,
		Address: request.Address,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, pickuppointdto.FromModel(point))
}

// @Summary     Удалить ПВЗ
// @Description Доступно только администратору. ПВЗ, который используется объявлениями, удалить нельзя.
// @Tags        admin pickup-points
// @Produce     json
// @Param       id path string true "UUID ПВЗ"
// @Success     204 "ПВЗ удалён"
// @Failure     400 {object} pickuppointdto.PickupPointError "Некорректный UUID"
// @Failure     401 {object} pickuppointdto.PickupPointError "Пользователь не авторизован"
// @Failure     403 {object} pickuppointdto.PickupPointError "Недостаточно прав"
// @Failure     404 {object} pickuppointdto.PickupPointError "ПВЗ не найден"
// @Failure     409 {object} pickuppointdto.PickupPointError "ПВЗ используется объявлениями"
// @Failure     500 {object} pickuppointdto.PickupPointError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /admin/pickup-points/{id} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pickup point id")
		return uuid.Nil, false
	}

	return id, true
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
	case errors.Is(err, pickuppointservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, pickuppointservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "pickup point not found")
	case errors.Is(err, pickuppointservice.ErrInUse):
		writeError(w, http.StatusConflict, "pickup point is used by items")
	default:
		log.Printf("pickup point handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, pickuppointdto.PickupPointError{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}
