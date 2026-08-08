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
	reportdto "github.com/sweetlife999/chain-of-trades-avito/internal/report/dto"
	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
	reportservice "github.com/sweetlife999/chain-of-trades-avito/internal/report/service"
)

const maxRequestBodyBytes = 1 << 20

type Service interface {
	Create(context.Context, reportservice.CreateInput) (reportmodel.Report, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router chi.Router, requireAuth func(http.Handler) http.Handler) {
	router.With(requireAuth).Post("/reports", h.create)
}

// @Summary     Пожаловаться на сообщение
// @Description Принимает жалобу на сообщение треда обмена. Пожаловаться можно только на чужое
// @Description сообщение участника в обмене, где жалобщик сам участвует, и только один раз.
// @Description На события обмена (подтверждение, отмена и прочие) жаловаться нельзя: их пишет сервис.
// @Description Жалоба попадает в очередь модерации со статусом open и ни на что в подборе не влияет.
// @Tags        reports
// @Accept      json
// @Produce     json
// @Param       request body reportdto.CreateReportRequest true "Сообщение, причина и необязательный комментарий"
// @Success     201 {object} reportdto.ReportResponse "Жалоба принята"
// @Failure     400 {object} reportdto.ReportError    "Некорректное тело, неизвестная причина, длинный комментарий или жалоба на событие обмена"
// @Failure     401 {object} reportdto.ReportError    "Пользователь не авторизован"
// @Failure     403 {object} reportdto.ReportError    "Не участник обмена или жалоба на собственное сообщение"
// @Failure     404 {object} reportdto.ReportError    "Сообщение не найдено"
// @Failure     409 {object} reportdto.ReportError    "Пользователь уже жаловался на это сообщение"
// @Failure     500 {object} reportdto.ReportError    "Внутренняя ошибка"
// @Security    CookieAuth
// @Router      /reports [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	reporterID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var request reportdto.CreateReportRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	messageID, err := uuid.Parse(request.MessageID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message id")
		return
	}

	report, err := h.service.Create(r.Context(), reportservice.CreateInput{
		ReporterID: reporterID,
		MessageID:  messageID,
		Reason:     request.Reason,
		Comment:    request.Comment,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, reportdto.FromModel(report))
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
	case errors.Is(err, reportservice.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, reportservice.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, reportservice.ErrNotFound):
		writeError(w, http.StatusNotFound, "message not found")
	case errors.Is(err, reportservice.ErrDuplicate):
		writeError(w, http.StatusConflict, "message is already reported")
	default:
		log.Printf("report handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, reportdto.ReportError{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}
