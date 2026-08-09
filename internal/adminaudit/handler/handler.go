package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	adminauditdto "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/dto"
	adminauditmodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/model"
	adminauditservice "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/service"
	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
)

type Service interface {
	BlockUser(context.Context, uuid.UUID, uuid.UUID) (adminauditmodel.UserBlockState, error)
	UnblockUser(context.Context, uuid.UUID, uuid.UUID) (adminauditmodel.UserBlockState, error)
	List(context.Context, adminauditmodel.Filter) (adminauditmodel.Page, error)
}

type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/users/{id}/block", h.blockUser)
	router.Post("/users/{id}/unblock", h.unblockUser)
	router.Get("/audit-log", h.list)
}

// @Summary     Глобально заблокировать пользователя
// @Description Блокировка сразу запрещает новый вход и использование ранее выданного JWT. Личный список user_blocks не изменяется.
// @Tags        admin-audit
// @Produce     json
// @Param       id path string true "UUID пользователя"
// @Success     200 {object} adminauditdto.UserBlockResponse
// @Failure     400 {object} adminauditdto.ErrorResponse
// @Failure     401 {object} adminauditdto.ErrorResponse
// @Failure     403 {object} adminauditdto.ErrorResponse
// @Failure     404 {object} adminauditdto.ErrorResponse
// @Failure     409 {object} adminauditdto.ErrorResponse
// @Security    CookieAuth
// @Router      /admin/users/{id}/block [post]
func (h *Handler) blockUser(w http.ResponseWriter, r *http.Request) {
	h.changeUserBlock(w, r, h.service.BlockUser)
}

// @Summary     Снять глобальную блокировку пользователя
// @Tags        admin-audit
// @Produce     json
// @Param       id path string true "UUID пользователя"
// @Success     200 {object} adminauditdto.UserBlockResponse
// @Failure     400 {object} adminauditdto.ErrorResponse
// @Failure     401 {object} adminauditdto.ErrorResponse
// @Failure     403 {object} adminauditdto.ErrorResponse
// @Failure     404 {object} adminauditdto.ErrorResponse
// @Failure     409 {object} adminauditdto.ErrorResponse
// @Security    CookieAuth
// @Router      /admin/users/{id}/unblock [post]
func (h *Handler) unblockUser(w http.ResponseWriter, r *http.Request) {
	h.changeUserBlock(w, r, h.service.UnblockUser)
}

func (h *Handler) changeUserBlock(w http.ResponseWriter, r *http.Request, change func(context.Context, uuid.UUID, uuid.UUID) (adminauditmodel.UserBlockState, error)) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	adminID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	state, err := change(r.Context(), adminID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, adminauditdto.UserBlockFromModel(state))
}

// @Summary     История действий администраторов
// @Description Возвращает журнал действий с фильтрами по администратору, действию и диапазону дат RFC3339.
// @Tags        admin-audit
// @Produce     json
// @Param       admin_id query string false "UUID администратора"
// @Param       action query string false "Тип действия"
// @Param       from query string false "Начало периода RFC3339"
// @Param       to query string false "Конец периода RFC3339"
// @Param       limit query int false "Размер страницы (1-100)" default(20)
// @Param       offset query int false "Смещение" default(0)
// @Success     200 {object} adminauditdto.PageResponse
// @Failure     400 {object} adminauditdto.ErrorResponse
// @Failure     401 {object} adminauditdto.ErrorResponse
// @Failure     403 {object} adminauditdto.ErrorResponse
// @Security    CookieAuth
// @Router      /admin/audit-log [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	filter, err := filterFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := h.service.List(r.Context(), filter)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, adminauditdto.PageFromModel(page))
}

func filterFromRequest(r *http.Request) (adminauditmodel.Filter, error) {
	query := r.URL.Query()
	filter := adminauditmodel.Filter{Action: query.Get("action")}
	if value := query.Get("admin_id"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			return filter, errors.New("invalid admin_id")
		}
		filter.AdminID = &id
	}
	var err error
	filter.From, err = optionalRFC3339(query.Get("from"))
	if err != nil {
		return filter, errors.New("invalid from")
	}
	filter.To, err = optionalRFC3339(query.Get("to"))
	if err != nil {
		return filter, errors.New("invalid to")
	}
	filter.Limit, err = int32Query(query.Get("limit"), adminauditservice.DefaultLimit)
	if err != nil {
		return filter, errors.New("invalid limit")
	}
	filter.Offset, err = int32Query(query.Get("offset"), 0)
	if err != nil {
		return filter, errors.New("invalid offset")
	}
	return filter, nil
}

func optionalRFC3339(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func int32Query(value string, fallback int32) (int32, error) {
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	return int32(parsed), err
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, adminauditservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, adminauditservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, adminauditservice.ErrConflict):
		writeError(w, http.StatusConflict, "user block state is unchanged")
	default:
		log.Printf("admin audit handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, adminauditdto.ErrorResponse{Error: message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode admin audit response: %v", err)
	}
}
