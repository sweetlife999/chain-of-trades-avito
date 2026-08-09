//go:build integration

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	adminauditrepository "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/repository"
	adminauditservice "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/service"
	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
)

func TestAdminUserBlockAuditIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	adminID, userID := uuid.New(), uuid.New()
	t.Cleanup(func() {
		cleanup := context.Background()
		_, _ = pool.Exec(cleanup, "DELETE FROM admin_audit_log WHERE admin_id = $1", adminID)
		_, _ = pool.Exec(cleanup, "DELETE FROM users WHERE id = ANY($1)", []uuid.UUID{adminID, userID})
		pool.Close()
	})
	for id, nickname := range map[uuid.UUID]string{adminID: "audit-admin", userID: "audit-user"} {
		if _, err := pool.Exec(ctx, "INSERT INTO users (id, nickname, password_hash, is_admin) VALUES ($1, $2, 'test', $3)", id, nickname+id.String()[:8], id == adminID); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	queries := db.New(pool)
	service := adminauditservice.New(adminauditrepository.New(queries))
	router := chi.NewRouter()
	New(service).RegisterRoutes(router)
	request := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, nil)
		r = r.WithContext(authcontext.WithUserID(r.Context(), adminID))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}

	blocked := request("/users/" + userID.String() + "/block")
	if blocked.Code != http.StatusOK {
		t.Fatalf("block status = %d, body = %s", blocked.Code, blocked.Body.String())
	}
	allowed, err := userrepository.New(queries).CanAuthenticate(ctx, userID)
	if err != nil || allowed {
		t.Fatalf("CanAuthenticate() = %v, %v; want false", allowed, err)
	}
	var blockAudits int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM admin_audit_log WHERE admin_id=$1 AND target_id=$2 AND action='user_blocked'", adminID, userID).Scan(&blockAudits); err != nil || blockAudits != 1 {
		t.Fatalf("block audits = %d, %v", blockAudits, err)
	}

	unblocked := request("/users/" + userID.String() + "/unblock")
	if unblocked.Code != http.StatusOK {
		t.Fatalf("unblock status = %d, body = %s", unblocked.Code, unblocked.Body.String())
	}
	allowed, err = userrepository.New(queries).CanAuthenticate(ctx, userID)
	if err != nil || !allowed {
		t.Fatalf("CanAuthenticate() = %v, %v; want true", allowed, err)
	}
}
