package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	userdto "github.com/sweetlife999/chain-of-trades-avito/internal/user/dto"
)

const CookieName = "access_token"

type TokenParser interface {
	Parse(string) (uuid.UUID, error)
}

type Authenticator struct {
	tokens TokenParser
}

func New(tokens TokenParser) *Authenticator {
	return &Authenticator{tokens: tokens}
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

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(userdto.ErrorResponse{Error: "unauthorized"})
}
