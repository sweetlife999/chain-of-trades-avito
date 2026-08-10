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
	supportdto "github.com/sweetlife999/chain-of-trades-avito/internal/support/dto"
	supportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/support/model"
	supportservice "github.com/sweetlife999/chain-of-trades-avito/internal/support/service"
)

const maxRequestBodyBytes = 1 << 20

type Service interface {
	Create(context.Context, uuid.UUID, string, string) (supportmodel.Thread, error)
	List(context.Context, uuid.UUID) ([]supportmodel.Thread, error)
	Messages(context.Context, uuid.UUID, uuid.UUID) (supportmodel.Thread, []supportmodel.Message, error)
	Send(context.Context, uuid.UUID, uuid.UUID, string) (supportmodel.Message, error)
	Close(context.Context, uuid.UUID, uuid.UUID) (supportmodel.Thread, error)
}

type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(
	router chi.Router,
	requireAuth func(http.Handler) http.Handler,
) {
	router.With(requireAuth).Get("/support/threads", h.list)
	router.With(requireAuth).Post("/support/threads", h.create)
	router.With(requireAuth).Get("/support/threads/{id}/messages", h.messages)
	router.With(requireAuth).Post("/support/threads/{id}/messages", h.send)
	router.With(requireAuth).Post("/support/threads/{id}/close", h.close)
}

// @Summary     Создать обращение в поддержку
// @Description Создаёт отдельный от обменов диалог. Одновременно разрешено одно активное обращение.
// @Tags        support
// @Accept      json
// @Produce     json
// @Param       request body supportdto.CreateThreadRequest true "Тема и первое сообщение"
// @Success     201 {object} supportdto.ThreadResponse
// @Failure     400 {object} supportdto.ErrorResponse
// @Failure     401 {object} supportdto.ErrorResponse
// @Failure     409 {object} supportdto.ErrorResponse "Уже есть активное обращение"
// @Failure     500 {object} supportdto.ErrorResponse
// @Security    CookieAuth
// @Router      /support/threads [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := requestUserID(w, r)
	if !ok {
		return
	}
	var request supportdto.CreateThreadRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	thread, err := h.service.Create(r.Context(), userID, request.Subject, request.Body)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, supportdto.ThreadFromModel(thread))
}

// @Summary  Мои обращения в поддержку
// @Tags     support
// @Produce  json
// @Success  200 {array} supportdto.ThreadResponse
// @Failure  401 {object} supportdto.ErrorResponse
// @Failure  500 {object} supportdto.ErrorResponse
// @Security CookieAuth
// @Router   /support/threads [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := requestUserID(w, r)
	if !ok {
		return
	}
	threads, err := h.service.List(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, supportdto.ThreadsFromModels(threads))
}

// @Summary  Переписка с поддержкой
// @Tags     support
// @Produce  json
// @Param    id path string true "UUID обращения"
// @Success  200 {object} supportdto.ThreadMessagesResponse
// @Failure  400 {object} supportdto.ErrorResponse
// @Failure  401 {object} supportdto.ErrorResponse
// @Failure  404 {object} supportdto.ErrorResponse
// @Failure  500 {object} supportdto.ErrorResponse
// @Security CookieAuth
// @Router   /support/threads/{id}/messages [get]
func (h *Handler) messages(w http.ResponseWriter, r *http.Request) {
	userID, ok := requestUserID(w, r)
	if !ok {
		return
	}
	threadID, ok := requestThreadID(w, r)
	if !ok {
		return
	}
	thread, messages, err := h.service.Messages(r.Context(), threadID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, supportdto.MessagesResponse(thread, messages))
}

// @Summary  Написать в поддержку
// @Tags     support
// @Accept   json
// @Produce  json
// @Param    id path string true "UUID обращения"
// @Param    request body supportdto.CreateMessageRequest true "Текст сообщения"
// @Success  201 {object} supportdto.MessageResponse
// @Failure  400 {object} supportdto.ErrorResponse
// @Failure  401 {object} supportdto.ErrorResponse
// @Failure  404 {object} supportdto.ErrorResponse
// @Failure  409 {object} supportdto.ErrorResponse "Обращение закрыто"
// @Failure  500 {object} supportdto.ErrorResponse
// @Security CookieAuth
// @Router   /support/threads/{id}/messages [post]
func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	userID, ok := requestUserID(w, r)
	if !ok {
		return
	}
	threadID, ok := requestThreadID(w, r)
	if !ok {
		return
	}
	var request supportdto.CreateMessageRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	message, err := h.service.Send(r.Context(), threadID, userID, request.Body)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, supportdto.MessageFromModel(message))
}

// @Summary  Закрыть обращение
// @Tags     support
// @Produce  json
// @Param    id path string true "UUID обращения"
// @Success  200 {object} supportdto.ThreadResponse
// @Failure  400 {object} supportdto.ErrorResponse
// @Failure  401 {object} supportdto.ErrorResponse
// @Failure  404 {object} supportdto.ErrorResponse
// @Failure  409 {object} supportdto.ErrorResponse
// @Failure  500 {object} supportdto.ErrorResponse
// @Security CookieAuth
// @Router   /support/threads/{id}/close [post]
func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
	userID, ok := requestUserID(w, r)
	if !ok {
		return
	}
	threadID, ok := requestThreadID(w, r)
	if !ok {
		return
	}
	thread, err := h.service.Close(r.Context(), threadID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, supportdto.ThreadFromModel(thread))
}

func requestUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
	}
	return userID, ok
}

func requestThreadID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid thread id")
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
	case errors.Is(err, supportservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, supportservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "support thread not found")
	case errors.Is(err, supportservice.ErrActiveExists):
		writeError(w, http.StatusConflict, "active support thread already exists")
	case errors.Is(err, supportservice.ErrConflict):
		writeError(w, http.StatusConflict, "support thread state conflict")
	default:
		log.Printf("support handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, supportdto.ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode support response: %v", err)
	}
}
