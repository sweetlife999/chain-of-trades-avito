package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authmiddleware "github.com/sweetlife999/chain-of-trades-avito/internal/auth/middleware"
	pickuppointdto "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/dto"
	pickuppointmodel "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/model"
	pickuppointservice "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/service"
)

type fakeService struct {
	create func(context.Context, pickuppointservice.CreateInput) (pickuppointmodel.PickupPoint, error)
	get    func(context.Context, uuid.UUID) (pickuppointmodel.PickupPoint, error)
	list   func(context.Context) ([]pickuppointmodel.PickupPoint, error)
	update func(context.Context, uuid.UUID, pickuppointservice.UpdateInput) (pickuppointmodel.PickupPoint, error)
	delete func(context.Context, uuid.UUID) error
}

func (f *fakeService) Create(ctx context.Context, input pickuppointservice.CreateInput) (pickuppointmodel.PickupPoint, error) {
	return f.create(ctx, input)
}

func (f *fakeService) GetByID(ctx context.Context, id uuid.UUID) (pickuppointmodel.PickupPoint, error) {
	return f.get(ctx, id)
}

func (f *fakeService) List(ctx context.Context) ([]pickuppointmodel.PickupPoint, error) {
	return f.list(ctx)
}

func (f *fakeService) Update(ctx context.Context, id uuid.UUID, input pickuppointservice.UpdateInput) (pickuppointmodel.PickupPoint, error) {
	return f.update(ctx, id, input)
}

func (f *fakeService) Delete(ctx context.Context, id uuid.UUID) error {
	return f.delete(ctx, id)
}

func newRouter(service Service) http.Handler {
	router := chi.NewRouter()
	New(service).RegisterRoutes(router)
	return router
}

func TestCreatePickupPoint(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	createdAt := time.Now().UTC()
	service := &fakeService{
		create: func(_ context.Context, input pickuppointservice.CreateInput) (pickuppointmodel.PickupPoint, error) {
			if input.Name != "ПВЗ Центр" || input.Address != "ул. Ленина, 10" {
				t.Fatalf("input = %+v", input)
			}
			return pickuppointmodel.PickupPoint{
				ID: id, Name: input.Name, Address: input.Address,
				CreatedAt: createdAt, UpdatedAt: createdAt,
			}, nil
		},
	}

	request := httptest.NewRequest(http.MethodPost, "/pickup-points", bytes.NewBufferString(`{"name":"ПВЗ Центр","address":"ул. Ленина, 10"}`))
	response := httptest.NewRecorder()
	newRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/admin/pickup-points/"+id.String() {
		t.Fatalf("Location = %q", location)
	}
	var body pickuppointdto.PickupPointResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID != id.String() || body.Name != "ПВЗ Центр" {
		t.Fatalf("body = %+v", body)
	}
}

func TestListPickupPointsReturnsJSONArray(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		list: func(context.Context) ([]pickuppointmodel.PickupPoint, error) {
			return nil, nil
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/pickup-points", nil)
	response := httptest.NewRecorder()
	newRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "[]\n" {
		t.Fatalf("status = %d, body = %q; want 200 and []", response.Code, response.Body.String())
	}
}

func TestGetUpdateAndDeletePickupPoint(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	newAddress := "Новый адрес"
	service := &fakeService{
		get: func(_ context.Context, actualID uuid.UUID) (pickuppointmodel.PickupPoint, error) {
			if actualID != id {
				t.Fatalf("get id = %v, want %v", actualID, id)
			}
			return pickuppointmodel.PickupPoint{ID: id, Name: "ПВЗ", Address: "Адрес"}, nil
		},
		update: func(_ context.Context, actualID uuid.UUID, input pickuppointservice.UpdateInput) (pickuppointmodel.PickupPoint, error) {
			if actualID != id || input.Address == nil || *input.Address != newAddress || input.Name != nil {
				t.Fatalf("update id = %v, input = %+v", actualID, input)
			}
			return pickuppointmodel.PickupPoint{ID: id, Name: "ПВЗ", Address: *input.Address}, nil
		},
		delete: func(_ context.Context, actualID uuid.UUID) error {
			if actualID != id {
				t.Fatalf("delete id = %v, want %v", actualID, id)
			}
			return nil
		},
	}
	router := newRouter(service)

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{name: "get", method: http.MethodGet, wantStatus: http.StatusOK},
		{name: "update", method: http.MethodPatch, body: `{"address":"Новый адрес"}`, wantStatus: http.StatusOK},
		{name: "delete", method: http.MethodDelete, wantStatus: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/pickup-points/"+id.String(), bytes.NewBufferString(test.body))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestPickupPointErrors(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "invalid id", method: http.MethodGet, path: "/pickup-points/not-a-uuid", wantStatus: http.StatusBadRequest},
		{name: "invalid json", method: http.MethodPost, path: "/pickup-points", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "not found", method: http.MethodGet, path: "/pickup-points/" + id.String(), serviceErr: pickuppointservice.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "in use", method: http.MethodDelete, path: "/pickup-points/" + id.String(), serviceErr: pickuppointservice.ErrInUse, wantStatus: http.StatusConflict},
		{name: "internal", method: http.MethodGet, path: "/pickup-points/" + id.String(), serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{
				create: func(context.Context, pickuppointservice.CreateInput) (pickuppointmodel.PickupPoint, error) {
					return pickuppointmodel.PickupPoint{}, test.serviceErr
				},
				get: func(context.Context, uuid.UUID) (pickuppointmodel.PickupPoint, error) {
					return pickuppointmodel.PickupPoint{}, test.serviceErr
				},
				delete: func(context.Context, uuid.UUID) error { return test.serviceErr },
			}
			request := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
			response := httptest.NewRecorder()
			newRouter(service).ServeHTTP(response, request)
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

func TestPickupPointRoutesRequireAdmin(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	serviceCalled := false
	service := &fakeService{
		list: func(context.Context) ([]pickuppointmodel.PickupPoint, error) {
			serviceCalled = true
			return nil, nil
		},
	}

	tests := []struct {
		name       string
		cookie     bool
		admin      bool
		wantStatus int
	}{
		{name: "unauthenticated", wantStatus: http.StatusUnauthorized},
		{name: "regular user", cookie: true, wantStatus: http.StatusForbidden},
		{name: "administrator", cookie: true, admin: true, wantStatus: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serviceCalled = false
			router := chi.NewRouter()
			authenticator := authmiddleware.New(fakeTokenParser{userID: userID})
			authorizer := authmiddleware.NewAdminAuthorizer(fakeAdminChecker{admin: test.admin})
			router.Route("/admin", func(admin chi.Router) {
				admin.Use(authenticator.RequireAuthentication)
				admin.Use(authorizer.RequireAdmin)
				New(service).RegisterRoutes(admin)
			})

			request := httptest.NewRequest(http.MethodGet, "/admin/pickup-points", nil)
			if test.cookie {
				request.AddCookie(&http.Cookie{Name: authmiddleware.CookieName, Value: "valid-token"})
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if serviceCalled != test.admin {
				t.Fatalf("serviceCalled = %v, want %v", serviceCalled, test.admin)
			}
		})
	}
}
