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
	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
	userservice "github.com/sweetlife999/chain-of-trades-avito/internal/user/service"
)

type fakeService struct {
	create      func(context.Context, userservice.CreateInput) (usermodel.User, error)
	get         func(context.Context, uuid.UUID) (usermodel.User, error)
	update      func(context.Context, uuid.UUID, userservice.UpdateInput) (usermodel.User, error)
	block       func(context.Context, uuid.UUID, uuid.UUID) error
	listBlocked func(context.Context, uuid.UUID) ([]usermodel.BlockedUser, error)
	unblock     func(context.Context, uuid.UUID, uuid.UUID) error
}

func (f *fakeService) Create(ctx context.Context, input userservice.CreateInput) (usermodel.User, error) {
	return f.create(ctx, input)
}

func (f *fakeService) GetByID(ctx context.Context, id uuid.UUID) (usermodel.User, error) {
	return f.get(ctx, id)
}

func (f *fakeService) Update(ctx context.Context, id uuid.UUID, input userservice.UpdateInput) (usermodel.User, error) {
	return f.update(ctx, id, input)
}

func (f *fakeService) Block(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	if f.block == nil {
		return nil
	}
	return f.block(ctx, blockerID, blockedID)
}

func (f *fakeService) ListBlocked(ctx context.Context, blockerID uuid.UUID) ([]usermodel.BlockedUser, error) {
	if f.listBlocked == nil {
		return nil, nil
	}
	return f.listBlocked(ctx, blockerID)
}

func (f *fakeService) Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	if f.unblock == nil {
		return nil
	}
	return f.unblock(ctx, blockerID, blockedID)
}

func TestCreateReturns201AndSafeUser(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	service := &fakeService{
		create: func(_ context.Context, input userservice.CreateInput) (usermodel.User, error) {
			return usermodel.User{
				ID:          id,
				Nickname:    input.Nickname,
				Description: input.Description,
				CreatedAt:   time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC),
			}, nil
		},
	}

	response := performRequest(service, http.MethodPost, "/users", `{
		"nickname":"Samir",
		"password":"password123",
		"description":"hello"
	}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if response.Header().Get("Location") != "/users/"+id.String() {
		t.Fatalf("Location = %q", response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Fatalf("response exposes password data: %s", response.Body.String())
	}
}

func TestGetReturns200(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	service := &fakeService{
		get: func(_ context.Context, actualID uuid.UUID) (usermodel.User, error) {
			if actualID != id {
				t.Fatalf("GetByID() id = %v, want %v", actualID, id)
			}
			return usermodel.User{ID: id, Nickname: "Samir"}, nil
		},
	}

	response := performRequest(service, http.MethodGet, "/users/"+id.String(), "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestUpdateReturns200(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	service := &fakeService{
		update: func(_ context.Context, actualID uuid.UUID, input userservice.UpdateInput) (usermodel.User, error) {
			return usermodel.User{ID: actualID, Nickname: *input.Nickname}, nil
		},
	}

	response := performRequest(service, http.MethodPatch, "/users/"+id.String(), `{"nickname":"NewName"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestUpdateReturns401WithoutAuthenticatedUser(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		update: func(context.Context, uuid.UUID, userservice.UpdateInput) (usermodel.User, error) {
			t.Fatal("Update() must not be called")
			return usermodel.User{}, nil
		},
	}

	response := performRequestWithAuth(
		service,
		http.MethodPatch,
		"/users/"+uuid.New().String(),
		`{"description":"new description"}`,
		passThroughAuth,
	)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
}

func TestUpdateReturns403ForDifferentUser(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		update: func(context.Context, uuid.UUID, userservice.UpdateInput) (usermodel.User, error) {
			t.Fatal("Update() must not be called")
			return usermodel.User{}, nil
		},
	}

	response := performRequestWithAuth(
		service,
		http.MethodPatch,
		"/users/"+uuid.New().String(),
		`{"description":"new description"}`,
		authenticateAs(uuid.New()),
	)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestBlockRoutes(t *testing.T) {
	t.Parallel()

	currentUserID := uuid.New()
	otherUserID := uuid.New()
	blockedAt := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
	service := &fakeService{
		block: func(_ context.Context, blockerID, blockedID uuid.UUID) error {
			if blockerID != currentUserID || blockedID != otherUserID {
				t.Fatalf("Block() args = (%s, %s), want (%s, %s)", blockerID, blockedID, currentUserID, otherUserID)
			}
			return nil
		},
		listBlocked: func(_ context.Context, blockerID uuid.UUID) ([]usermodel.BlockedUser, error) {
			if blockerID != currentUserID {
				t.Fatalf("ListBlocked() user = %s, want %s", blockerID, currentUserID)
			}
			return []usermodel.BlockedUser{{ID: otherUserID, Nickname: "blocked-user", BlockedAt: blockedAt}}, nil
		},
		unblock: func(_ context.Context, blockerID, blockedID uuid.UUID) error {
			if blockerID != currentUserID || blockedID != otherUserID {
				t.Fatalf("Unblock() args = (%s, %s), want (%s, %s)", blockerID, blockedID, currentUserID, otherUserID)
			}
			return nil
		},
	}
	authenticate := authenticateAs(currentUserID)

	blockResponse := performRequestWithAuth(service, http.MethodPost, "/users/me/blocks/"+otherUserID.String(), "", authenticate)
	if blockResponse.Code != http.StatusNoContent {
		t.Fatalf("block status = %d, want %d; body = %s", blockResponse.Code, http.StatusNoContent, blockResponse.Body.String())
	}

	listResponse := performRequestWithAuth(service, http.MethodGet, "/users/me/blocks", "", authenticate)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body = %s", listResponse.Code, http.StatusOK, listResponse.Body.String())
	}
	if !strings.Contains(listResponse.Body.String(), otherUserID.String()) || !strings.Contains(listResponse.Body.String(), "blocked_at") {
		t.Fatalf("list response does not contain blocked user: %s", listResponse.Body.String())
	}

	unblockResponse := performRequestWithAuth(service, http.MethodDelete, "/users/me/blocks/"+otherUserID.String(), "", authenticate)
	if unblockResponse.Code != http.StatusNoContent {
		t.Fatalf("unblock status = %d, want %d; body = %s", unblockResponse.Code, http.StatusNoContent, unblockResponse.Body.String())
	}
}

func TestBlockRoutesRequireAuthentication(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		block: func(context.Context, uuid.UUID, uuid.UUID) error {
			t.Fatal("Block() must not be called")
			return nil
		},
		listBlocked: func(context.Context, uuid.UUID) ([]usermodel.BlockedUser, error) {
			t.Fatal("ListBlocked() must not be called")
			return nil, nil
		},
		unblock: func(context.Context, uuid.UUID, uuid.UUID) error {
			t.Fatal("Unblock() must not be called")
			return nil
		},
	}
	otherID := uuid.New().String()
	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/users/me/blocks"},
		{method: http.MethodPost, path: "/users/me/blocks/" + otherID},
		{method: http.MethodDelete, path: "/users/me/blocks/" + otherID},
	}

	for _, test := range tests {
		response := performRequestWithAuth(service, test.method, test.path, "", passThroughAuth)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want %d", test.method, test.path, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestBlockRouteRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	response := performRequestWithAuth(
		&fakeService{},
		http.MethodPost,
		"/users/me/blocks/not-a-uuid",
		"",
		authenticateAs(uuid.New()),
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestHandlerErrorStatuses(t *testing.T) {
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
		{name: "invalid JSON", method: http.MethodPost, path: "/users", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "unknown JSON field", method: http.MethodPost, path: "/users", body: `{"unknown":true}`, wantStatus: http.StatusBadRequest},
		{name: "invalid UUID", method: http.MethodGet, path: "/users/not-a-uuid", wantStatus: http.StatusBadRequest},
		{name: "not found", method: http.MethodGet, path: "/users/" + id.String(), serviceErr: userservice.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "nickname conflict", method: http.MethodPost, path: "/users", body: `{"nickname":"Samir","password":"password123"}`, serviceErr: userservice.ErrNicknameTaken, wantStatus: http.StatusConflict},
		{name: "internal error", method: http.MethodGet, path: "/users/" + id.String(), serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeService{
				create: func(context.Context, userservice.CreateInput) (usermodel.User, error) {
					return usermodel.User{}, test.serviceErr
				},
				get: func(context.Context, uuid.UUID) (usermodel.User, error) {
					return usermodel.User{}, test.serviceErr
				},
				update: func(context.Context, uuid.UUID, userservice.UpdateInput) (usermodel.User, error) {
					return usermodel.User{}, test.serviceErr
				},
			}

			response := performRequest(service, test.method, test.path, test.body)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func performRequest(service Service, method, path, body string) *httptest.ResponseRecorder {
	return performRequestWithAuth(service, method, path, body, authenticateAsPathUser)
}

func performRequestWithAuth(
	service Service,
	method string,
	path string,
	body string,
	requireAuth func(http.Handler) http.Handler,
) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	New(service).RegisterRoutes(router, requireAuth)

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func authenticateAs(userID uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(authcontext.WithUserID(r.Context(), userID))
			next.ServeHTTP(w, r)
		})
	}
}

func passThroughAuth(next http.Handler) http.Handler {
	return next
}

func authenticateAsPathUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err == nil {
			r = r.WithContext(authcontext.WithUserID(r.Context(), userID))
		}
		next.ServeHTTP(w, r)
	})
}
