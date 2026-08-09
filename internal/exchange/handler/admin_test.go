package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
)

func TestAdminCancelExchange(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	service := &fakeService{adminCancel: func(_ context.Context, actualID, adminID uuid.UUID) error {
		if actualID != exchangeID {
			t.Fatalf("CancelByAdmin() exchange ID = %s, want %s", actualID, exchangeID)
		}
		if adminID == uuid.Nil {
			t.Fatal("CancelByAdmin() admin ID is empty")
		}
		return nil
	}}

	response := performAdminRequest(service, "/exchanges/"+exchangeID.String()+"/cancel")
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNoContent, response.Body.String())
	}
}

func TestAdminCancelExchangeErrorStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pathID     string
		serviceErr error
		wantStatus int
	}{
		{name: "invalid UUID", pathID: "not-a-uuid", wantStatus: http.StatusBadRequest},
		{name: "not found", pathID: uuid.New().String(), serviceErr: exchangeservice.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict", pathID: uuid.New().String(), serviceErr: exchangeservice.ErrConflict, wantStatus: http.StatusConflict},
		{name: "internal", pathID: uuid.New().String(), serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{adminCancel: func(context.Context, uuid.UUID, uuid.UUID) error {
				return test.serviceErr
			}}
			response := performAdminRequest(service, "/exchanges/"+test.pathID+"/cancel")
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func performAdminRequest(service Service, path string) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	New(service).RegisterAdminRoutes(router)

	request := httptest.NewRequest(http.MethodPost, path, nil)
	request = request.WithContext(authcontext.WithUserID(request.Context(), uuid.New()))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
