package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	authdto "github.com/sweetlife999/chain-of-trades-avito/internal/auth/dto"
	authmiddleware "github.com/sweetlife999/chain-of-trades-avito/internal/auth/middleware"
	authservice "github.com/sweetlife999/chain-of-trades-avito/internal/auth/service"
	userdto "github.com/sweetlife999/chain-of-trades-avito/internal/user/dto"
	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
)

const maxRequestBodyBytes = 1 << 20

type Service interface {
	Login(context.Context, authservice.LoginInput) (authservice.LoginResult, error)
	CurrentUser(context.Context, uuid.UUID) (usermodel.User, error)
}

type Handler struct {
	service      Service
	cookieSecure bool
	cookieTTL    time.Duration
}

func New(service Service, cookieSecure bool, cookieTTL time.Duration) *Handler {
	return &Handler{
		service:      service,
		cookieSecure: cookieSecure,
		cookieTTL:    cookieTTL,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	router.Route("/auth", func(router chi.Router) {
		router.Post("/login", h.login)
		router.Post("/logout", h.logout)
		router.With(requireAuth).Get("/me", h.me)
	})
}

// @Summary     Вход
// @Description Проверяет nickname и password, кладёт JWT на 12 часов в HttpOnly cookie `access_token`. Из Swagger UI cookie сохранится в браузере, дальше защищённые запросы уйдут с ней автоматически.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body     authdto.LoginRequest  true "Учётные данные"
// @Success     200     {object} authdto.AuthenticatedUserResponse "Вошедший пользователь и его роль, cookie в заголовке Set-Cookie"
// @Failure     400     {object} userdto.ErrorResponse "Некорректное тело запроса"
// @Failure     401     {object} userdto.ErrorResponse "Неверный nickname или пароль"
// @Failure     403     {object} userdto.ErrorResponse "Аккаунт глобально заблокирован"
// @Failure     500     {object} userdto.ErrorResponse "Внутренняя ошибка"
// @Router      /auth/login [post]
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var request authdto.LoginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.Login(r.Context(), authservice.LoginInput{
		Nickname: request.Nickname,
		Password: request.Password,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	h.setAccessCookie(w, result.Token, result.ExpiresAt)
	writeJSON(w, http.StatusOK, authdto.UserFromModel(result.User))
}

// @Summary     Выход
// @Description Удаляет cookie `access_token`. Сам JWT остаётся валидным до истечения срока — серверного списка отзыва нет.
// @Tags        auth
// @Success     204 "Cookie удалена"
// @Router      /auth/logout [post]
func (h *Handler) logout(w http.ResponseWriter, _ *http.Request) {
	h.clearAccessCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// @Summary     Текущий пользователь
// @Description Возвращает профиль владельца cookie `access_token`. Удобно для проверки, что вход прошёл.
// @Tags        auth
// @Produce     json
// @Success     200 {object} authdto.AuthenticatedUserResponse "Профиль и роль текущего пользователя"
// @Failure     401 {object} userdto.ErrorResponse "Нет или истекла cookie access_token"
// @Failure     500 {object} userdto.ErrorResponse "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /auth/me [get]
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.CurrentUser(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authdto.UserFromModel(user))
}

func (h *Handler) setAccessCookie(w http.ResponseWriter, value string, expiresAt time.Time) {
	// #nosec G124 -- Secure берётся из COOKIE_SECURE, чтобы локальная разработка шла по http; статически gosec это не проверить.
	http.SetCookie(w, &http.Cookie{
		Name:     authmiddleware.CookieName,
		Value:    value,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(h.cookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearAccessCookie(w http.ResponseWriter) {
	// #nosec G124 -- то же, что и в setAccessCookie: значение Secure задаётся конфигом.
	http.SetCookie(w, &http.Cookie{
		Name:     authmiddleware.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
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
	case errors.Is(err, authservice.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid nickname or password")
	case errors.Is(err, authservice.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, authservice.ErrAccountBlocked):
		writeError(w, http.StatusForbidden, "account is blocked")
	default:
		log.Printf("auth handler: %v", err)
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
