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

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	itemdto "github.com/sweetlife999/chain-of-trades-avito/internal/item/dto"
	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
	itemservice "github.com/sweetlife999/chain-of-trades-avito/internal/item/service"
)

const maxRequestBodyBytes = 1 << 20

type Service interface {
	Create(context.Context, itemservice.CreateInput) (itemmodel.Item, error)
	GetByID(context.Context, uuid.UUID) (itemmodel.Item, error)
	ListByOwner(context.Context, uuid.UUID) ([]itemmodel.Item, error)
	Update(context.Context, uuid.UUID, uuid.UUID, itemservice.UpdateInput) (itemmodel.Item, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	ListCategories(context.Context) ([]itemmodel.Category, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	router.With(requireAuth).Post("/items", h.create)
	router.With(requireAuth).Get("/items", h.listMine)
	router.Get("/items/{id}", h.getByID)
	router.With(requireAuth).Patch("/items/{id}", h.update)
	router.With(requireAuth).Delete("/items/{id}", h.delete)
	router.Get("/categories", h.listCategories)
}

// @Summary     Создать объявление
// @Description Требует cookie `access_token`. Владелец берётся из токена, а не из тела запроса.
// @Description Нужна хотя бы одна фотография (ссылкой) и хотя бы одна желаемая категория —
// @Description без них объявление не участвует в подборе обменов. Список категорий: GET /categories.
// @Tags        items
// @Accept      json
// @Produce     json
// @Param       request body     itemdto.CreateItemRequest true "Данные объявления"
// @Success     201     {object} itemdto.ItemResponse "Создано, ссылка на объявление в заголовке Location"
// @Failure     400     {object} itemdto.ItemError    "Некорректное тело запроса, нет фото или желаний, неизвестная категория"
// @Failure     401     {object} itemdto.ItemError    "Нет или истекла cookie access_token"
// @Failure     500     {object} itemdto.ItemError    "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /items [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentUser(w, r)
	if !ok {
		return
	}

	var request itemdto.CreateItemRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Create(r.Context(), itemservice.CreateInput{
		OwnerID:     ownerID,
		Category:    request.Category,
		Title:       request.Title,
		Description: request.Description,
		PhotoURLs:   request.PhotoURLs,
		Wants:       request.Wants,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	w.Header().Set("Location", "/items/"+item.ID.String())
	writeJSON(w, http.StatusCreated, itemdto.FromModel(item))
}

// @Summary     Получить свои объявления
// @Description Требует cookie `access_token`. Возвращает все объявления текущего пользователя
// @Description любого статуса, от новых к старым. Чужие объявления этим маршрутом не отдаются.
// @Tags        items
// @Produce     json
// @Success     200 {array}  itemdto.ItemResponse "Список объявлений; если их нет, возвращается []"
// @Failure     401 {object} itemdto.ItemError    "Нет или истекла cookie access_token"
// @Failure     500 {object} itemdto.ItemError    "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /items [get]
func (h *Handler) listMine(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentUser(w, r)
	if !ok {
		return
	}

	items, err := h.service.ListByOwner(r.Context(), ownerID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, itemdto.ItemsFromModel(items))
}

// @Summary     Получить объявление по ID
// @Description Публичная карточка вещи: фотографии, желаемые категории и статус. Аутентификация не нужна.
// @Tags        items
// @Produce     json
// @Param       id  path     string                true "UUID объявления" example(8db9f3e2-8a45-4a70-b3d1-167b4f97e121)
// @Success     200 {object} itemdto.ItemResponse "Объявление"
// @Failure     400 {object} itemdto.ItemError    "ID не является UUID"
// @Failure     404 {object} itemdto.ItemError    "Объявление не найдено"
// @Failure     500 {object} itemdto.ItemError    "Внутренняя ошибка"
// @Router      /items/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	item, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, itemdto.FromModel(item))
}

// @Summary     Изменить своё объявление
// @Description Требует cookie `access_token`, менять можно только свои объявления. Достаточно одного поля.
// @Description `photo_urls` и `wants` заменяются целиком: чтобы добавить фотографию, пришлите старые
// @Description ссылки вместе с новой. Пустой список запрещён — у объявления всегда есть хотя бы одно фото.
// @Tags        items
// @Accept      json
// @Produce     json
// @Param       id      path     string                    true "UUID объявления" example(8db9f3e2-8a45-4a70-b3d1-167b4f97e121)
// @Param       request body     itemdto.UpdateItemRequest true "Поля, которые нужно изменить"
// @Success     200     {object} itemdto.ItemResponse "Обновлённое объявление"
// @Failure     400     {object} itemdto.ItemError    "Некорректное тело, пустой список фото или желаний, неизвестная категория"
// @Failure     401     {object} itemdto.ItemError    "Нет или истекла cookie access_token"
// @Failure     403     {object} itemdto.ItemError    "Объявление принадлежит другому пользователю"
// @Failure     404     {object} itemdto.ItemError    "Объявление не найдено"
// @Failure     409     {object} itemdto.ItemError    "Нельзя изменить условия объявления в незавершённом обмене"
// @Failure     500     {object} itemdto.ItemError    "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /items/{id} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	var request itemdto.UpdateItemRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Update(r.Context(), id, userID, itemservice.UpdateInput{
		Category:    request.Category,
		Title:       request.Title,
		Description: request.Description,
		PhotoURLs:   request.PhotoURLs,
		Wants:       request.Wants,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, itemdto.FromModel(item))
}

// @Summary     Удалить своё объявление
// @Description Требует cookie `access_token`, удалять можно только свои объявления. Вещь, уже занятую
// @Description в незавершённом обмене, удалить нельзя: иначе участник остался бы без обещанного предмета.
// @Tags        items
// @Produce     json
// @Param       id path string true "UUID объявления" example(8db9f3e2-8a45-4a70-b3d1-167b4f97e121)
// @Success     204 "Удалено"
// @Failure     400 {object} itemdto.ItemError "ID не является UUID"
// @Failure     401 {object} itemdto.ItemError "Нет или истекла cookie access_token"
// @Failure     403 {object} itemdto.ItemError "Объявление принадлежит другому пользователю"
// @Failure     404 {object} itemdto.ItemError "Объявление не найдено"
// @Failure     409 {object} itemdto.ItemError "Вещь участвует в незавершённом обмене"
// @Failure     500 {object} itemdto.ItemError "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /items/{id} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	userID, ok := currentUser(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary     Список категорий
// @Description Справочник для полей `category` и `wants`. Слаг — стабильный ключ, название меняется, слаг нет.
// @Tags        items
// @Produce     json
// @Success     200 {array}  itemdto.CategoryResponse "Категории"
// @Failure     500 {object} itemdto.ItemError        "Внутренняя ошибка"
// @Router      /categories [get]
func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.ListCategories(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, itemdto.CategoriesFromModel(categories))
}

func currentUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}

	return userID, true
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return uuid.Nil, false
	}

	return id, true
}

// TODO: третья копия decodeJSON/writeJSON/writeError — вместе с auth и user хэндлерами
// просятся в общий пакет. Выносить сейчас — трогать чужие модули вне скоупа #13.
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
	case errors.Is(err, itemservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, itemservice.ErrUnknownCategory):
		writeError(w, http.StatusBadRequest, "unknown category")
	case errors.Is(err, itemservice.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, itemservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "item not found")
	case errors.Is(err, itemservice.ErrItemInChain):
		writeError(w, http.StatusConflict, "item participates in an open exchange")
	default:
		log.Printf("item handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, itemdto.ItemError{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}
