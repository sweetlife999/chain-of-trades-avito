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
	userdto "github.com/sweetlife999/chain-of-trades-avito/internal/user/dto"
	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
	userservice "github.com/sweetlife999/chain-of-trades-avito/internal/user/service"
)

const maxRequestBodyBytes = 1 << 20

type Service interface {
	Create(context.Context, userservice.CreateInput) (usermodel.User, error)
	GetByID(context.Context, uuid.UUID) (usermodel.User, error)
	Update(context.Context, uuid.UUID, userservice.UpdateInput) (usermodel.User, error)
	Block(context.Context, uuid.UUID, uuid.UUID) error
	ListBlocked(context.Context, uuid.UUID) ([]usermodel.BlockedUser, error)
	Unblock(context.Context, uuid.UUID, uuid.UUID) error
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	router.Post("/users", h.create)
	router.Get("/users/{id}", h.getByID)
	router.With(requireAuth).Patch("/users/{id}", h.update)
	router.With(requireAuth).Get("/users/me/blocks", h.listBlocked)
	router.With(requireAuth).Post("/users/me/blocks/{user_id}", h.block)
	router.With(requireAuth).Delete("/users/me/blocks/{user_id}", h.unblock)
}

// @Summary     Создать пользователя
// @Description Регистрация: nickname 3–32 символа, пароль 8–72 байта. Пароль хранится bcrypt-хешем и в ответе не возвращается.
// @Tags        users
// @Accept      json
// @Produce     json
// @Param       request body     userdto.CreateUserRequest true "Данные нового пользователя"
// @Success     201     {object} userdto.UserResponse      "Создан, ссылка на профиль в заголовке Location"
// @Failure     400     {object} userdto.ErrorResponse     "Некорректное тело запроса или нарушены ограничения полей"
// @Failure     409     {object} userdto.ErrorResponse     "Nickname уже занят"
// @Failure     500     {object} userdto.ErrorResponse     "Внутренняя ошибка"
// @Router      /users [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var request userdto.CreateUserRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Create(r.Context(), userservice.CreateInput{
		Nickname:    request.Nickname,
		Password:    request.Password,
		PhotoURL:    request.PhotoURL,
		Description: request.Description,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	w.Header().Set("Location", "/users/"+user.ID.String())
	writeJSON(w, http.StatusCreated, userdto.FromModel(user))
}

// @Summary     Получить пользователя по ID
// @Description Публичный профиль: рейтинг и счётчики сделок. Аутентификация не нужна.
// @Tags        users
// @Produce     json
// @Param       id  path     string                true "UUID пользователя" example(8db9f3e2-8a45-4a70-b3d1-167b4f97e121)
// @Success     200 {object} userdto.UserResponse  "Профиль пользователя"
// @Failure     400 {object} userdto.ErrorResponse "ID не является UUID"
// @Failure     404 {object} userdto.ErrorResponse "Пользователь не найден"
// @Failure     500 {object} userdto.ErrorResponse "Внутренняя ошибка"
// @Router      /users/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, userdto.FromModel(user))
}

// @Summary     Обновить свой профиль
// @Description Требует cookie `access_token` (получить через POST /auth/login). Менять можно только собственный профиль. Достаточно одного поля, остальные останутся прежними.
// @Tags        users
// @Accept      json
// @Produce     json
// @Param       id      path     string                    true "UUID пользователя" example(8db9f3e2-8a45-4a70-b3d1-167b4f97e121)
// @Param       request body     userdto.UpdateUserRequest true "Поля, которые нужно изменить"
// @Success     200     {object} userdto.UserResponse      "Обновлённый профиль"
// @Failure     400     {object} userdto.ErrorResponse     "Некорректное тело запроса, не передано ни одного поля или ID не UUID"
// @Failure     401     {object} userdto.ErrorResponse     "Нет или истекла cookie access_token"
// @Failure     403     {object} userdto.ErrorResponse     "Попытка изменить чужой профиль"
// @Failure     404     {object} userdto.ErrorResponse     "Пользователь не найден"
// @Failure     409     {object} userdto.ErrorResponse     "Nickname уже занят"
// @Failure     500     {object} userdto.ErrorResponse     "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /users/{id} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	currentUserID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if currentUserID != id {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var request userdto.UpdateUserRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Update(r.Context(), id, userservice.UpdateInput{
		Nickname:    request.Nickname,
		PhotoURL:    request.PhotoURL,
		Description: request.Description,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, userdto.FromModel(user))
}

// @Summary     Получить список заблокированных пользователей
// @Description Возвращает личный список блокировок текущего пользователя. Заблокированные пользователи об этом не уведомляются.
// @Tags        users
// @Produce     json
// @Success     200 {array}  userdto.BlockedUserResponse "Список блокировок; если он пуст, возвращается []"
// @Failure     401 {object} userdto.ErrorResponse       "Пользователь не авторизован"
// @Failure     500 {object} userdto.ErrorResponse       "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /users/me/blocks [get]
func (h *Handler) listBlocked(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	blocked, err := h.service.ListBlocked(r.Context(), currentUserID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, userdto.BlockedFromModels(blocked))
}

// @Summary     Заблокировать пользователя
// @Description Исключает пользователя из будущих совместных обменов и отменяет общие неподтверждённые предложения. Повторный запрос безопасен.
// @Tags        users
// @Param       user_id path string true "UUID блокируемого пользователя"
// @Success     204 "Пользователь заблокирован"
// @Failure     400 {object} userdto.ErrorResponse "ID не является UUID или пользователь блокирует себя"
// @Failure     401 {object} userdto.ErrorResponse "Пользователь не авторизован"
// @Failure     404 {object} userdto.ErrorResponse "Блокируемый пользователь не найден"
// @Failure     500 {object} userdto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /users/me/blocks/{user_id} [post]
func (h *Handler) block(w http.ResponseWriter, r *http.Request) {
	h.handleBlockChange(w, r, h.service.Block)
}

// @Summary     Разблокировать пользователя
// @Description Разрешает попадание пользователя в новые совместные обмены. Существующие обмены не изменяются.
// @Tags        users
// @Param       user_id path string true "UUID разблокируемого пользователя"
// @Success     204 "Пользователь разблокирован"
// @Failure     400 {object} userdto.ErrorResponse "ID не является UUID или пользователь указывает себя"
// @Failure     401 {object} userdto.ErrorResponse "Пользователь не авторизован"
// @Failure     500 {object} userdto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /users/me/blocks/{user_id} [delete]
func (h *Handler) unblock(w http.ResponseWriter, r *http.Request) {
	h.handleBlockChange(w, r, h.service.Unblock)
}

func (h *Handler) handleBlockChange(
	w http.ResponseWriter,
	r *http.Request,
	change func(context.Context, uuid.UUID, uuid.UUID) error,
) {
	currentUserID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	blockedID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := change(r.Context(), currentUserID, blockedID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
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
		writeError(w, http.StatusBadRequest, "invalid user id")
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
	case errors.Is(err, userservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, userservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, userservice.ErrNicknameTaken):
		writeError(w, http.StatusConflict, "nickname is already taken")
	default:
		log.Printf("user handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, userdto.ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}
