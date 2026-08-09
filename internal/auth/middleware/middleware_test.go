package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
)

type fakeTokenParser struct {
	userID uuid.UUID
	err    error
}

type fakeAdminChecker struct {
	check func(context.Context, uuid.UUID) (bool, error)
}

type fakeAccountChecker struct {
	check func(context.Context, uuid.UUID) (bool, error)
}

func (f *fakeAccountChecker) CanAuthenticate(ctx context.Context, userID uuid.UUID) (bool, error) {
	return f.check(ctx, userID)
}

func (f *fakeAdminChecker) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	return f.check(ctx, userID)
}

func (f *fakeTokenParser) Parse(string) (uuid.UUID, error) {
	return f.userID, f.err
}

func TestRequireAuthenticationAddsUserIDToContext(t *testing.T) {
	t.Parallel()

	wantID := uuid.New()
	authenticator := New(&fakeTokenParser{userID: wantID}, &fakeAccountChecker{
		check: func(context.Context, uuid.UUID) (bool, error) { return true, nil },
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualID, ok := authcontext.UserID(r.Context())
		if !ok || actualID != wantID {
			t.Fatalf("context user id = %v, %v; want %v, true", actualID, ok, wantID)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: CookieName, Value: "valid-token"})
	response := httptest.NewRecorder()
	authenticator.RequireAuthentication(next).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestRequireAuthenticationReturns401(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		parser *fakeTokenParser
		cookie *http.Cookie
	}{
		{name: "missing cookie", parser: &fakeTokenParser{}},
		{
			name:   "invalid token",
			parser: &fakeTokenParser{err: errors.New("invalid token")},
			cookie: &http.Cookie{Name: CookieName, Value: "invalid-token"},
		},
		{
			name:   "empty cookie",
			parser: &fakeTokenParser{},
			cookie: &http.Cookie{Name: CookieName, Value: ""},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.cookie != nil {
				request.AddCookie(test.cookie)
			}
			response := httptest.NewRecorder()
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("protected handler must not be called")
			})

			New(test.parser, &fakeAccountChecker{
				check: func(context.Context, uuid.UUID) (bool, error) { return true, nil },
			}).RequireAuthentication(next).ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
			}
		})
	}
}

func TestRequireAuthenticationRejectsBlockedAccountAndOldToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	authenticator := New(
		&fakeTokenParser{userID: userID},
		&fakeAccountChecker{check: func(_ context.Context, actualID uuid.UUID) (bool, error) {
			if actualID != userID {
				t.Fatalf("account check id = %v, want %v", actualID, userID)
			}
			return false, nil
		}},
	)
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: CookieName, Value: "previously-issued-token"})
	response := httptest.NewRecorder()
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler must not be called")
	})

	authenticator.RequireAuthentication(next).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestRequireAdminAllowsAdministrator(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	authorizer := NewAdminAuthorizer(&fakeAdminChecker{
		check: func(_ context.Context, actualID uuid.UUID) (bool, error) {
			if actualID != userID {
				t.Fatalf("admin check id = %v, want %v", actualID, userID)
			}
			return true, nil
		},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	request = request.WithContext(authcontext.WithUserID(request.Context(), userID))
	response := httptest.NewRecorder()

	authorizer.RequireAdmin(next).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNoContent, response.Body.String())
	}
}

func TestRequireAdminRejectsRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		withUserID bool
		check      func(context.Context, uuid.UUID) (bool, error)
		wantStatus int
	}{
		{
			name:       "missing authenticated user",
			withUserID: false,
			check: func(context.Context, uuid.UUID) (bool, error) {
				t.Fatal("admin checker must not be called")
				return false, nil
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "regular user",
			withUserID: true,
			check: func(context.Context, uuid.UUID) (bool, error) {
				return false, nil
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin lookup failure",
			withUserID: true,
			check: func(context.Context, uuid.UUID) (bool, error) {
				return false, errors.New("database unavailable")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if test.withUserID {
				request = request.WithContext(authcontext.WithUserID(request.Context(), uuid.New()))
			}
			response := httptest.NewRecorder()
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("protected handler must not be called")
			})

			NewAdminAuthorizer(&fakeAdminChecker{check: test.check}).
				RequireAdmin(next).
				ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}
