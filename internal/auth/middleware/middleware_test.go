package middleware

import (
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

func (f *fakeTokenParser) Parse(string) (uuid.UUID, error) {
	return f.userID, f.err
}

func TestRequireAuthenticationAddsUserIDToContext(t *testing.T) {
	t.Parallel()

	wantID := uuid.New()
	authenticator := New(&fakeTokenParser{userID: wantID})
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

			New(test.parser).RequireAuthentication(next).ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
			}
		})
	}
}
