package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	admindashboarddto "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/dto"
	admindashboardmodel "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/model"
	authmiddleware "github.com/sweetlife999/chain-of-trades-avito/internal/auth/middleware"
)

type fakeService struct {
	dashboard admindashboardmodel.Dashboard
	err       error
	called    bool
}

func (f *fakeService) Get(context.Context) (admindashboardmodel.Dashboard, error) {
	f.called = true
	return f.dashboard, f.err
}

func TestGetDashboard(t *testing.T) {
	t.Parallel()

	service := &fakeService{dashboard: admindashboardmodel.Dashboard{
		UsersTotal:        4,
		PickupPointsTotal: 2,
		Items:             admindashboardmodel.ItemStatistics{Total: 9, Available: 6},
		Exchanges:         admindashboardmodel.ExchangeStatistics{Total: 3, Completed: 1},
	}}
	router := chi.NewRouter()
	New(service).RegisterRoutes(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/dashboard", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var body admindashboarddto.DashboardResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.UsersTotal != 4 || body.Items.Available != 6 || body.Exchanges.Completed != 1 {
		t.Fatalf("body = %+v", body)
	}
}

func TestGetDashboardHandlesServiceError(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	New(&fakeService{err: errors.New("database unavailable")}).RegisterRoutes(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/dashboard", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

type fakeTokenParser struct {
	userID uuid.UUID
}

func (f fakeTokenParser) Parse(string) (uuid.UUID, error) {
	return f.userID, nil
}

type fakeAdminChecker struct {
	admin bool
}

func (f fakeAdminChecker) IsAdmin(context.Context, uuid.UUID) (bool, error) {
	return f.admin, nil
}

func (f fakeAdminChecker) CanAuthenticate(context.Context, uuid.UUID) (bool, error) {
	return true, nil
}

func TestDashboardRouteRequiresAdministrator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cookie     bool
		admin      bool
		wantStatus int
		wantCall   bool
	}{
		{name: "unauthenticated", wantStatus: http.StatusUnauthorized},
		{name: "regular user", cookie: true, wantStatus: http.StatusForbidden},
		{name: "administrator", cookie: true, admin: true, wantStatus: http.StatusOK, wantCall: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{}
			userID := uuid.New()
			checker := fakeAdminChecker{admin: test.admin}
			authenticator := authmiddleware.New(fakeTokenParser{userID: userID}, checker)
			authorizer := authmiddleware.NewAdminAuthorizer(checker)
			router := chi.NewRouter()
			router.Route("/admin", func(admin chi.Router) {
				admin.Use(authenticator.RequireAuthentication)
				admin.Use(authorizer.RequireAdmin)
				New(service).RegisterRoutes(admin)
			})

			request := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
			if test.cookie {
				request.AddCookie(&http.Cookie{Name: authmiddleware.CookieName, Value: "valid-token"})
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if service.called != test.wantCall {
				t.Fatalf("service called = %v, want %v", service.called, test.wantCall)
			}
		})
	}
}
