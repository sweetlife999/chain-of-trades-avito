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

	antiscammodel "github.com/sweetlife999/chain-of-trades-avito/internal/antiscam/model"
	antiscamservice "github.com/sweetlife999/chain-of-trades-avito/internal/antiscam/service"
	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	exchangedto "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/dto"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

type Service interface {
	List(context.Context, antiscammodel.Filter) (antiscammodel.Page, error)
	Get(context.Context, uuid.UUID) (antiscammodel.Case, error)
	Messages(context.Context, uuid.UUID) (antiscammodel.Case, []exchangemodel.Message, []uuid.UUID, error)
	Decide(context.Context, uuid.UUID, uuid.UUID, string, string) (antiscammodel.Case, error)
}

type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/antiscam/cases", h.list)
	router.Get("/antiscam/cases/{id}", h.get)
	router.Get("/antiscam/cases/{id}/messages", h.messages)
	router.Post("/antiscam/cases/{id}/confirm", h.confirm)
	router.Post("/antiscam/cases/{id}/dismiss", h.dismiss)
}

type userResponse struct {
	ID       string  `json:"id"`
	Nickname string  `json:"nickname"`
	PhotoURL *string `json:"photo_url"`
}
type evidenceResponse struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
type caseResponse struct {
	ID                string           `json:"id"`
	ExchangeID        string           `json:"exchange_id"`
	Suspect           userResponse     `json:"suspect"`
	Status            string           `json:"status"`
	Risk              int32            `json:"risk"`
	Category          string           `json:"category"`
	Reason            string           `json:"reason"`
	Reviewer          *userResponse    `json:"reviewer"`
	Decision          *string          `json:"decision"`
	ResolutionComment string           `json:"resolution_comment"`
	LatestEvidence    evidenceResponse `json:"latest_evidence"`
	EvidenceCount     int64            `json:"evidence_count"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	ClosedAt          *time.Time       `json:"closed_at" extensions:"x-nullable"`
}
type pageResponse struct {
	Cases      []caseResponse `json:"cases"`
	Pagination struct {
		Limit  int32 `json:"limit"`
		Offset int32 `json:"offset"`
		Total  int64 `json:"total"`
	} `json:"pagination"`
}
type messagesResponse struct {
	CaseID             string                        `json:"case_id"`
	ExchangeID         string                        `json:"exchange_id"`
	EvidenceMessageIDs []string                      `json:"evidence_message_ids"`
	Messages           []exchangedto.MessageResponse `json:"messages"`
}
type decisionRequest struct {
	Comment string `json:"comment"`
}
type errorResponse struct {
	Error string `json:"error"`
}

// @Summary Очередь AI-антискама
// @Tags admin-antiscam
// @Produce json
// @Param status query string false "Статус" Enums(open,resolved,dismissed)
// @Param category query string false "Категория"
// @Param min_risk query int false "Минимальный риск" default(0)
// @Param limit query int false "Размер страницы" default(20)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} pageResponse
// @Security CookieAuth
// @Router /admin/antiscam/cases [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, err := int32Value(query.Get("limit"), antiscamservice.DefaultAdminLimit)
	if err != nil {
		writeError(w, 400, "invalid limit")
		return
	}
	offset, err := int32Value(query.Get("offset"), 0)
	if err != nil {
		writeError(w, 400, "invalid offset")
		return
	}
	minRisk, err := int32Value(query.Get("min_risk"), 0)
	if err != nil {
		writeError(w, 400, "invalid min_risk")
		return
	}
	page, err := h.service.List(r.Context(), antiscammodel.Filter{Status: query.Get("status"), Category: query.Get("category"), MinRisk: minRisk, Limit: limit, Offset: offset})
	if err != nil {
		handleError(w, err)
		return
	}
	response := pageResponse{Cases: make([]caseResponse, len(page.Cases))}
	response.Pagination.Limit = page.Limit
	response.Pagination.Offset = page.Offset
	response.Pagination.Total = page.Total
	for index, item := range page.Cases {
		response.Cases[index] = fromModel(item)
	}
	writeJSON(w, 200, response)
}

// @Summary Карточка AI-подозрения
// @Tags admin-antiscam
// @Produce json
// @Param id path string true "UUID карточки"
// @Success 200 {object} caseResponse
// @Security CookieAuth
// @Router /admin/antiscam/cases/{id} [get]
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := caseID(w, r)
	if !ok {
		return
	}
	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, 200, fromModel(item))
}

// @Summary Переписка по AI-подозрению
// @Tags admin-antiscam
// @Produce json
// @Param id path string true "UUID карточки"
// @Success 200 {object} messagesResponse
// @Security CookieAuth
// @Router /admin/antiscam/cases/{id}/messages [get]
func (h *Handler) messages(w http.ResponseWriter, r *http.Request) {
	id, ok := caseID(w, r)
	if !ok {
		return
	}
	item, messages, evidence, err := h.service.Messages(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	evidenceStrings := make([]string, len(evidence))
	for i, value := range evidence {
		evidenceStrings[i] = value.String()
	}
	writeJSON(w, 200, messagesResponse{CaseID: id.String(), ExchangeID: item.ExchangeID.String(), EvidenceMessageIDs: evidenceStrings, Messages: exchangedto.MessagesFromModels(messages)})
}

// @Summary Подтвердить AI-подозрение
// @Tags admin-antiscam
// @Accept json
// @Produce json
// @Param id path string true "UUID карточки"
// @Param request body decisionRequest true "Комментарий"
// @Success 200 {object} caseResponse
// @Security CookieAuth
// @Router /admin/antiscam/cases/{id}/confirm [post]
func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, antiscammodel.DecisionConfirmed)
}

// @Summary Отметить ложное AI-срабатывание
// @Tags admin-antiscam
// @Accept json
// @Produce json
// @Param id path string true "UUID карточки"
// @Param request body decisionRequest true "Комментарий"
// @Success 200 {object} caseResponse
// @Security CookieAuth
// @Router /admin/antiscam/cases/{id}/dismiss [post]
func (h *Handler) dismiss(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, antiscammodel.DecisionFalsePositive)
}

func (h *Handler) decide(w http.ResponseWriter, r *http.Request, decision string) {
	id, ok := caseID(w, r)
	if !ok {
		return
	}
	adminID, ok := authcontext.UserID(r.Context())
	if !ok {
		writeError(w, 401, "unauthorized")
		return
	}
	var request decisionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	item, err := h.service.Decide(r.Context(), id, adminID, decision, request.Comment)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, 200, fromModel(item))
}

func fromModel(item antiscammodel.Case) caseResponse {
	response := caseResponse{ID: item.ID.String(), ExchangeID: item.ExchangeID.String(), Suspect: userResponse{ID: item.Suspect.ID.String(), Nickname: item.Suspect.Nickname, PhotoURL: item.Suspect.PhotoURL}, Status: item.Status, Risk: item.Risk, Category: item.Category, Reason: item.Reason, Decision: item.Decision, ResolutionComment: item.ResolutionComment, LatestEvidence: evidenceResponse{ID: item.LatestEvidence.ID.String(), Body: item.LatestEvidence.Body, CreatedAt: item.LatestEvidence.CreatedAt}, EvidenceCount: item.EvidenceCount, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, ClosedAt: item.ClosedAt}
	if item.Reviewer != nil {
		response.Reviewer = &userResponse{ID: item.Reviewer.ID.String(), Nickname: item.Reviewer.Nickname, PhotoURL: item.Reviewer.PhotoURL}
	}
	return response
}
func caseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 400, "invalid case id")
		return uuid.Nil, false
	}
	return id, true
}
func int32Value(value string, fallback int32) (int32, error) {
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	return int32(parsed), err
}
func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, antiscamservice.ErrValidation):
		writeError(w, 400, err.Error())
	case errors.Is(err, antiscamservice.ErrNotFound):
		writeError(w, 404, "antiscam case not found")
	case errors.Is(err, antiscamservice.ErrAlreadyReviewed):
		writeError(w, 409, "antiscam case is already reviewed")
	default:
		log.Printf("antiscam handler: %v", err)
		writeError(w, 500, "internal server error")
	}
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode antiscam response: %v", err)
	}
}
