package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	itemassistant "github.com/sweetlife999/chain-of-trades-avito/internal/itemassistant/service"
)

const maxRequestBodyBytes = 8 << 10

type Service interface {
	Submit(uuid.UUID, string) (itemassistant.Job, error)
	Get(uuid.UUID, uuid.UUID) (itemassistant.Job, error)
}

type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	router.With(requireAuth).Post("/items/ai-suggestions", h.submit)
	router.With(requireAuth).Get("/items/ai-suggestions/{id}", h.get)
}

type SubmitRequest struct {
	Input string `json:"input" example:"старый пленочный фотоаппарат, рабочий, есть чехол"`
}

type SuggestionResponse struct {
	Title        string `json:"title" example:"Плёночный фотоаппарат с чехлом"`
	Description  string `json:"description" example:"Рабочий плёночный фотоаппарат. В комплекте чехол."`
	CategorySlug string `json:"category_slug" example:"electronics"`
	CategoryName string `json:"category_name" example:"Электроника"`
}

type JobResponse struct {
	ID         string              `json:"id"`
	Status     string              `json:"status" enums:"pending,processing,completed,failed"`
	Suggestion *SuggestionResponse `json:"suggestion,omitempty"`
	Error      string              `json:"error,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// @Summary     Запустить ИИ-помощника объявления
// @Description Принимает рассказ пользователя, ставит генерацию в фоновую очередь и сразу возвращает ID задачи.
// @Description Результат не применяется автоматически: пользователь должен проверить и принять подсказку на фронте.
// @Tags        items
// @Accept      json
// @Produce     json
// @Param       request body SubmitRequest true "Описание вещи своими словами"
// @Success     202 {object} JobResponse
// @Failure     400 {object} ErrorResponse "Текст короче 10 или длиннее 1200 символов"
// @Failure     401 {object} ErrorResponse "Нет или истекла cookie access_token"
// @Failure     429 {object} ErrorResponse "Очередь модели заполнена"
// @Security    CookieAuth
// @Router      /items/ai-suggestions [post]
func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "AI assistant is disabled")
		return
	}
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var request SubmitRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	job, err := h.service.Submit(userID, request.Input)
	if err != nil {
		switch {
		case errors.Is(err, itemassistant.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, itemassistant.ErrQueueFull):
			writeError(w, http.StatusTooManyRequests, "AI assistant is busy, try again later")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusAccepted, fromJob(job))
}

// @Summary     Получить результат ИИ-помощника
// @Description Задача доступна только создавшему её пользователю и хранится 30 минут.
// @Tags        items
// @Produce     json
// @Param       id path string true "UUID задачи"
// @Success     200 {object} JobResponse
// @Failure     400 {object} ErrorResponse "Некорректный UUID"
// @Failure     401 {object} ErrorResponse "Нет или истекла cookie access_token"
// @Failure     403 {object} ErrorResponse "Чужая задача"
// @Failure     404 {object} ErrorResponse "Задача не найдена или истекла"
// @Security    CookieAuth
// @Router      /items/ai-suggestions/{id} [get]
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "AI assistant is disabled")
		return
	}
	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid suggestion job id")
		return
	}
	job, err := h.service.Get(userID, id)
	if err != nil {
		switch {
		case errors.Is(err, itemassistant.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, itemassistant.ErrNotFound):
			writeError(w, http.StatusNotFound, "suggestion job not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, fromJob(job))
}

func fromJob(job itemassistant.Job) JobResponse {
	response := JobResponse{ID: job.ID.String(), Status: string(job.Status), Error: job.Error, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt}
	if job.Suggestion != nil {
		response.Suggestion = &SuggestionResponse{
			Title: job.Suggestion.Title, Description: job.Suggestion.Description,
			CategorySlug: job.Suggestion.CategorySlug, CategoryName: job.Suggestion.CategoryName,
		}
	}
	return response
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

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
