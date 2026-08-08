package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	reportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/report/model"
	reportservice "github.com/sweetlife999/chain-of-trades-avito/internal/report/service"
)

type fakeService struct {
	report reportmodel.Report
	err    error
}

func (f *fakeService) Create(context.Context, reportservice.CreateInput) (reportmodel.Report, error) {
	return f.report, f.err
}

func request(t *testing.T, body string, userID *uuid.UUID, service Service) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	New(service).RegisterRoutes(router, func(next http.Handler) http.Handler { return next })

	r := httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(body))
	if userID != nil {
		r = r.WithContext(authcontext.WithUserID(r.Context(), *userID))
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	return w
}

func TestCreateReportStatusCodes(t *testing.T) {
	t.Parallel()

	body := `{"message_id":"` + uuid.New().String() + `","reason":"abuse","comment":""}`

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "принята", want: http.StatusCreated},
		{name: "валидация", err: reportservice.ErrValidation, want: http.StatusBadRequest},
		{name: "запрещено", err: reportservice.ErrForbidden, want: http.StatusForbidden},
		{name: "сообщения нет", err: reportservice.ErrNotFound, want: http.StatusNotFound},
		{name: "повтор", err: reportservice.ErrDuplicate, want: http.StatusConflict},
		{name: "неизвестная ошибка", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			userID := uuid.New()
			w := request(t, body, &userID, &fakeService{err: test.err})
			if w.Code != test.want {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, test.want, w.Body.String())
			}
		})
	}
}

func TestCreateReportRejectsBadRequests(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name string
		body string
	}{
		{name: "битый JSON", body: `{"message_id":`},
		{name: "message_id не UUID", body: `{"message_id":"not-a-uuid","reason":"spam"}`},
		{name: "лишнее поле", body: `{"message_id":"` + uuid.New().String() + `","status":"resolved"}`},
		{name: "два объекта", body: `{"message_id":"x"}{"message_id":"y"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w := request(t, test.body, &userID, &fakeService{})
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestCreateReportRequiresUser(t *testing.T) {
	t.Parallel()

	body := `{"message_id":"` + uuid.New().String() + `","reason":"spam"}`
	if w := request(t, body, nil, &fakeService{}); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
