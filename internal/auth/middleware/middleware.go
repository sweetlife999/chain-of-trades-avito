package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	userdto "github.com/sweetlife999/chain-of-trades-avito/internal/user/dto"
)

const CookieName = "access_token"

type TokenParser interface {
	Parse(string) (uuid.UUID, error)
}

type AdminChecker interface {
	IsAdmin(context.Context, uuid.UUID) (bool, error)
}

type Authenticator struct {
	tokens TokenParser
}

func New(tokens TokenParser) *Authenticator {
	return &Authenticator{tokens: tokens}
}

type AdminAuthorizer struct {
	users AdminChecker
}

func NewAdminAuthorizer(users AdminChecker) *AdminAuthorizer {
	return &AdminAuthorizer{users: users}
}

func (a *Authenticator) RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil || cookie.Value == "" {
			writeUnauthorized(w)
			return
		}

		userID, err := a.tokens.Parse(cookie.Value)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		ctx := authcontext.WithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin запускается после RequireAuthentication. Роль читается из БД, а не
// из JWT: выданное или отозванное право действует сразу, даже для старой cookie.
func (a *AdminAuthorizer) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authcontext.UserID(r.Context())
		if !ok {
			writeUnauthorized(w)
			return
		}

		isAdmin, err := a.users.IsAdmin(r.Context(), userID)
		if err != nil {
			log.Printf("admin authorization: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if !isAdmin {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeUnauthorized(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, "unauthorized")
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(userdto.ErrorResponse{Error: message})
}
