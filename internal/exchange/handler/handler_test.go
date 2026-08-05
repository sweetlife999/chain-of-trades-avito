package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
)

func TestListReturnsExchanges(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	want := exchangeDetails(userID)
	service := &fakeService{list: func(_ context.Context, actualUserID uuid.UUID) ([]exchangemodel.Details, error) {
		if actualUserID != userID {
			t.Fatalf("ListForUser() user ID = %s, want %s", actualUserID, userID)
		}
		return []exchangemodel.Details{want}, nil
	}}

	response := performRequest(service, http.MethodGet, "/exchanges", authenticateAs(userID))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	body := response.Body.String()
	for _, field := range []string{"participants", "gives_item", "receives_item", "nickname"} {
		if !strings.Contains(body, `"`+field+`"`) {
			t.Fatalf("response does not contain %q: %s", field, body)
		}
	}

	if strings.Contains(body, `"chain`) {
		t.Fatalf("response exposes internal chain terminology: %s", body)
	}
}

func TestListReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	service := &fakeService{list: func(context.Context, uuid.UUID) ([]exchangemodel.Details, error) {
		return nil, nil
	}}

	response := performRequest(service, http.MethodGet, "/exchanges", authenticateAs(uuid.New()))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if response.Body.String() != "[]\n" {
		t.Fatalf("body = %q, want []", response.Body.String())
	}
}

func TestGetByIDReturnsExchange(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	want := exchangeDetails(userID)
	service := &fakeService{get: func(
		_ context.Context,
		exchangeID uuid.UUID,
		actualUserID uuid.UUID,
	) (exchangemodel.Details, error) {
		if exchangeID != want.ID {
			t.Fatalf("GetForUser() exchange ID = %s, want %s", exchangeID, want.ID)
		}
		if actualUserID != userID {
			t.Fatalf("GetForUser() user ID = %s, want %s", actualUserID, userID)
		}
		return want, nil
	}}

	response := performRequest(
		service,
		http.MethodGet,
		"/exchanges/"+want.ID.String(),
		authenticateAs(userID),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	if !strings.Contains(response.Body.String(), want.ID.String()) {
		t.Fatalf("response does not contain exchange ID: %s", response.Body.String())
	}
}

func TestRoutesRequireAuthentication(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		list: func(context.Context, uuid.UUID) ([]exchangemodel.Details, error) {
			t.Fatal("ListForUser() must not be called")
			return nil, nil
		},
		get: func(context.Context, uuid.UUID, uuid.UUID) (exchangemodel.Details, error) {
			t.Fatal("GetForUser() must not be called")
			return exchangemodel.Details{}, nil
		},
	}

	for _, path := range []string{"/exchanges", "/exchanges/" + uuid.New().String()} {
		response := performRequest(service, http.MethodGet, path, passThroughAuth)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("path %s: status = %d, want %d", path, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestGetByIDErrorStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		serviceErr error
		wantStatus int
	}{
		{name: "invalid UUID", path: "/exchanges/not-a-uuid", wantStatus: http.StatusBadRequest},
		{name: "forbidden", path: "/exchanges/" + uuid.New().String(), serviceErr: exchangeservice.ErrForbidden, wantStatus: http.StatusForbidden},
		{name: "not found", path: "/exchanges/" + uuid.New().String(), serviceErr: exchangeservice.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "internal error", path: "/exchanges/" + uuid.New().String(), serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{get: func(
				context.Context,
				uuid.UUID,
				uuid.UUID,
			) (exchangemodel.Details, error) {
				return exchangemodel.Details{}, test.serviceErr
			}}

			response := performRequest(service, http.MethodGet, test.path, authenticateAs(uuid.New()))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestListInternalError(t *testing.T) {
	t.Parallel()

	service := &fakeService{list: func(context.Context, uuid.UUID) ([]exchangemodel.Details, error) {
		return nil, errors.New("database unavailable")
	}}

	response := performRequest(service, http.MethodGet, "/exchanges", authenticateAs(uuid.New()))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

type fakeService struct {
	list func(context.Context, uuid.UUID) ([]exchangemodel.Details, error)
	get  func(context.Context, uuid.UUID, uuid.UUID) (exchangemodel.Details, error)
}

func (f *fakeService) ListForUser(
	ctx context.Context,
	userID uuid.UUID,
) ([]exchangemodel.Details, error) {
	return f.list(ctx, userID)
}

func (f *fakeService) GetForUser(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) (exchangemodel.Details, error) {
	return f.get(ctx, exchangeID, userID)
}

func performRequest(
	service Service,
	method string,
	path string,
	requireAuth func(http.Handler) http.Handler,
) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	New(service).RegisterRoutes(router, requireAuth)

	request := httptest.NewRequest(method, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func authenticateAs(userID uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(authcontext.WithUserID(r.Context(), userID)))
		})
	}
}

func passThroughAuth(next http.Handler) http.Handler {
	return next
}

func exchangeDetails(userID uuid.UUID) exchangemodel.Details {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	return exchangemodel.Details{
		ID:        uuid.New(),
		Status:    "proposed",
		CreatedAt: now,
		UpdatedAt: now,
		Participants: []exchangemodel.DetailsParticipant{{
			User: exchangemodel.ParticipantUser{
				ID:       userID,
				Nickname: "samir",
			},
			GivesItem: exchangemodel.ParticipantItem{
				ID:     uuid.New(),
				Title:  "Book",
				Status: "available",
				Category: exchangemodel.ParticipantCategory{
					Slug: "books",
					Name: "Books",
				},
			},
			ReceivesItem: exchangemodel.ParticipantItem{
				ID:     uuid.New(),
				Title:  "Game",
				Status: "available",
				Category: exchangemodel.ParticipantCategory{
					Slug: "hobby",
					Name: "Hobby",
				},
			},
			Position: 0,
			Status:   "pending",
		}},
	}
}
