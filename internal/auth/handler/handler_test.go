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
	authservice "github.com/sweetlife999/chain-of-trades-avito/internal/auth/service"
	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
)

type fakeService struct {
	login       func(context.Context, authservice.LoginInput) (authservice.LoginResult, error)
	currentUser func(context.Context, uuid.UUID) (usermodel.User, error)
}

func (f *fakeService) Login(ctx context.Context, input authservice.LoginInput) (authservice.LoginResult, error) {
	return f.login(ctx, input)
}

func (f *fakeService) CurrentUser(ctx context.Context, userID uuid.UUID) (usermodel.User, error) {
	return f.currentUser(ctx, userID)
}

func TestLoginReturnsSafeUserAndHttpOnlyCookie(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	expiresAt := time.Now().Add(12 * time.Hour).UTC()
	service := &fakeService{
		login: func(_ context.Context, input authservice.LoginInput) (authservice.LoginResult, error) {
			if input.Nickname != "Samir" || input.Password != "password123" {
				t.Fatalf("Login() input = %#v", input)
			}
			return authservice.LoginResult{
				User: usermodel.User{
					ID:           userID,
					Nickname:     "Samir",
					PasswordHash: "must-not-be-returned",
				},
				Token:     "signed-token",
				ExpiresAt: expiresAt,
			}, nil
		},
	}

	response := performRequest(service, true, 12*time.Hour, http.MethodPost, "/auth/login", `{
		"nickname":"Samir",
		"password":"password123"
	}`, passThroughAuth)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Fatalf("response exposes password data: %s", response.Body.String())
	}

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "access_token" || cookie.Value != "signed-token" {
		t.Fatalf("cookie = %#v", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie security flags = %#v", cookie)
	}
	if cookie.MaxAge != int((12*time.Hour).Seconds()) || !cookie.Expires.Equal(expiresAt.Truncate(time.Second)) {
		t.Fatalf("cookie lifetime = %#v", cookie)
	}
}

func TestLoginReturns401ForInvalidCredentials(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		login: func(context.Context, authservice.LoginInput) (authservice.LoginResult, error) {
			return authservice.LoginResult{}, authservice.ErrInvalidCredentials
		},
	}

	response := performRequest(service, false, 12*time.Hour, http.MethodPost, "/auth/login", `{
		"nickname":"Samir",
		"password":"wrong-password"
	}`, passThroughAuth)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
	if len(response.Result().Cookies()) != 0 {
		t.Fatal("invalid login must not create a cookie")
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	t.Parallel()

	response := performRequest(&fakeService{}, false, 12*time.Hour, http.MethodPost, "/auth/logout", "", passThroughAuth)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies count = %d, want 1", len(cookies))
	}
	if cookies[0].Value != "" || cookies[0].MaxAge >= 0 || !cookies[0].Expires.Before(time.Now()) {
		t.Fatalf("logout cookie = %#v", cookies[0])
	}
}

func TestMeReturnsCurrentUserWithoutPasswordHash(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	service := &fakeService{
		currentUser: func(_ context.Context, actualID uuid.UUID) (usermodel.User, error) {
			if actualID != userID {
				t.Fatalf("CurrentUser() id = %v, want %v", actualID, userID)
			}
			return usermodel.User{
				ID:           userID,
				Nickname:     "Samir",
				PasswordHash: "must-not-be-returned",
			}, nil
		},
	}

	response := performRequest(service, false, 12*time.Hour, http.MethodGet, "/auth/me", "", authenticateAs(userID))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Fatalf("response exposes password data: %s", response.Body.String())
	}
}

func TestMeReturns401WhenCurrentUserIsMissing(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		currentUser: func(context.Context, uuid.UUID) (usermodel.User, error) {
			return usermodel.User{}, authservice.ErrUnauthorized
		},
	}

	response := performRequest(service, false, 12*time.Hour, http.MethodGet, "/auth/me", "", authenticateAs(uuid.New()))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
}

func TestLoginRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	service := &fakeService{
		login: func(context.Context, authservice.LoginInput) (authservice.LoginResult, error) {
			return authservice.LoginResult{}, errors.New("must not be called")
		},
	}
	response := performRequest(service, false, 12*time.Hour, http.MethodPost, "/auth/login", `{"unknown":true}`, passThroughAuth)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func performRequest(
	service Service,
	cookieSecure bool,
	cookieTTL time.Duration,
	method string,
	path string,
	body string,
	requireAuth func(http.Handler) http.Handler,
) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	New(service, cookieSecure, cookieTTL).RegisterRoutes(router, requireAuth)

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
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
