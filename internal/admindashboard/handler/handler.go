package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	admindashboarddto "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/dto"
	admindashboardmodel "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/model"
)

type DashboardService interface {
	Get(context.Context) (admindashboardmodel.Dashboard, error)
}

type Handler struct {
	service DashboardService
}

func New(service DashboardService) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes вызывается только внутри уже защищённой группы /admin.
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/dashboard", h.get)
}

// get godoc
// @Summary     Статистика для главной страницы админки
// @Description Возвращает количество пользователей, ПВЗ, объявлений и обменов с разбивкой по статусам.
// @Tags        admin dashboard
// @Produce     json
// @Security    CookieAuth
// @Success     200 {object} admindashboarddto.DashboardResponse "Статистика админки"
// @Failure     401 {object} admindashboarddto.DashboardError "Пользователь не авторизован"
// @Failure     403 {object} admindashboarddto.DashboardError "Недостаточно прав"
// @Failure     500 {object} admindashboarddto.DashboardError "Внутренняя ошибка"
// @Router      /admin/dashboard [get]
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	dashboard, err := h.service.Get(r.Context())
	if err != nil {
		log.Printf("admin dashboard handler: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, admindashboarddto.FromModel(dashboard))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, admindashboarddto.DashboardError{Error: message})
}
