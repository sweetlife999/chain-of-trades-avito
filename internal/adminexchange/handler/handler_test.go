package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	adminexchangedto "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/dto"
	adminexchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/model"
	adminexchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/service"
	authmiddleware "github.com/sweetlife999/chain-of-trades-avito/internal/auth/middleware"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

type fakeService struct {
	page      adminexchangemodel.Page
	err       error
	called    bool
	gotUserID uuid.UUID
	gotLimit  int32
	gotOffset int32
}

func (f *fakeService) ListActiveByUser(
	_ context.Context,
	userID uuid.UUID,
	limit int32,
	offset int32,
) (adminexchangemodel.Page, error) {
	f.called = true
	f.gotUserID = userID
	f.gotLimit = limit
	f.gotOffset = offset
	return f.page, f.err
}

func newRouter(service Service) http.Handler {
	router := chi.NewRouter()
	New(service).RegisterRoutes(router)
	return router
}

func TestListActiveExchangesByUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	exchangeID := uuid.New()
	now := time.Now().UTC()
	service := &fakeService{page: adminexchangemodel.Page{
		Exchanges: []exchangemodel.Details{{
			ID: exchangeID, Status: "confirmed", CreatedAt: now, UpdatedAt: now,
		}},
		Limit: 10, Offset: 5, Total: 1,
	}}

	request := httptest.NewRequest(
		http.MethodGet,
		"/users/"+userID.String()+"/exchanges?limit=10&offset=5",
		nil,
	)
	response := httptest.NewRecorder()
	newRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	if service.gotUserID != userID || service.gotLimit != 10 || service.gotOffset != 5 {
		t.Fatalf("arguments = %v, %d, %d", service.gotUserID, service.gotLimit, service.gotOffset)
	}

	var body adminexchangedto.ListResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Exchanges) != 1 || body.Exchanges[0].ID != exchangeID.String() || body.Pagination.Total != 1 {
		t.Fatalf("body = %+v", body)
	}
}

func TestListActiveExchangesByUserUsesPaginationDefaults(t *testing.T) {
	t.Parallel()

	service := &fakeService{page: adminexchangemodel.Page{Exchanges: []exchangemodel.Details{}}}
	request := httptest.NewRequest(http.MethodGet, "/users/"+uuid.NewString()+"/exchanges", nil)
	response := httptest.NewRecorder()
	newRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body.String())
	}
	if service.gotLimit != adminexchangeservice.DefaultLimit || service.gotOffset != 0 {
		t.Fatalf("defaults = %d/%d", service.gotLimit, service.gotOffset)
	}
}

func TestListActiveExchangesByUserErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.NewString()
	tests := []struct {
		name       string
		path       string
		serviceErr error
		wantStatus int
	}{
		{name: "invalid user id", path: "/users/not-a-uuid/exchanges", wantStatus: http.StatusBadRequest},
		{name: "invalid limit", path: "/users/" + userID + "/exchanges?limit=text", wantStatus: http.StatusBadRequest},
		{name: "invalid offset", path: "/users/" + userID + "/exchanges?offset=text", wantStatus: http.StatusBadRequest},
		{name: "invalid pagination", path: "/users/" + userID + "/exchanges?limit=0", serviceErr: adminexchangeservice.ErrValidation, wantStatus: http.StatusBadRequest},
		{name: "user not found", path: "/users/" + userID + "/exchanges", serviceErr: adminexchangeservice.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "internal error", path: "/users/" + userID + "/exchanges", serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			newRouter(&fakeService{err: test.serviceErr}).ServeHTTP(
				response,
				httptest.NewRequest(http.MethodGet, test.path, nil),
			)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

type fakeTokenParser struct{ userID uuid.UUID }

func (f fakeTokenParser) Parse(string) (uuid.UUID, error) { return f.userID, nil }

type fakeAdminChecker struct{ admin bool }

func (f fakeAdminChecker) IsAdmin(context.Context, uuid.UUID) (bool, error) { return f.admin, nil }

func TestAdminExchangeRouteRequiresAdministrator(t *testing.T) {
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
			service := &fakeService{page: adminexchangemodel.Page{Exchanges: []exchangemodel.Details{}}}
			authenticator := authmiddleware.New(fakeTokenParser{userID: uuid.New()})
			authorizer := authmiddleware.NewAdminAuthorizer(fakeAdminChecker{admin: test.admin})
			router := chi.NewRouter()
			router.Route("/admin", func(admin chi.Router) {
				admin.Use(authenticator.RequireAuthentication)
				admin.Use(authorizer.RequireAdmin)
				New(service).RegisterRoutes(admin)
			})

			request := httptest.NewRequest(
				http.MethodGet,
				"/admin/users/"+uuid.NewString()+"/exchanges",
				nil,
			)
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
