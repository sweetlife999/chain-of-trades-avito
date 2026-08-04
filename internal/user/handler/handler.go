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
}

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
