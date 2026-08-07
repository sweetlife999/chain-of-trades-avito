//go:build integration

package handler

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
	itemrepository "github.com/sweetlife999/chain-of-trades-avito/internal/item/repository"
	itemservice "github.com/sweetlife999/chain-of-trades-avito/internal/item/service"
)

func TestThreeUserExchangeIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	users := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	items := make([]uuid.UUID, 0, 3)
	t.Cleanup(func() {
		cleanupIntegrationData(context.Background(), pool, users, items)
		pool.Close()
	})

	for index, userID := range users {
		_, err := pool.Exec(
			ctx,
			"INSERT INTO users (id, nickname, password_hash) VALUES ($1, $2, $3)",
			userID,
			"exchange-"+userID.String()[:8],
			"not-used-in-integration-test",
		)
		if err != nil {
			t.Fatalf("create user %d: %v", index, err)
		}
	}

	repository := exchangerepository.New(pool)
	service := exchangeservice.New(repository)
	itemsService := itemservice.New(itemrepository.New(pool), service)
	categories := []string{"books", "hobby", "sports"}
	wants := []string{"hobby", "sports", "books"}

	for index := range users {
		created, err := itemsService.Create(ctx, itemservice.CreateInput{
			OwnerID:   users[index],
			Category:  categories[index],
			Title:     "Integration item",
			PhotoURLs: []string{"https://example.com/integration.jpg"},
			Wants:     []string{wants[index]},
		})
		if err != nil {
			t.Fatalf("create item %d through service: %v", index, err)
		}
		items = append(items, created.ID)
	}

	found, err := service.ListForUser(ctx, users[0])
	if err != nil {
		t.Fatalf("list automatically found exchanges: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("automatically found exchanges = %d, want 1", len(found))
	}
	exchangeID := found[0].ID

	listResponse := performRequest(service, http.MethodGet, "/exchanges", authenticateAs(users[0]))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d; body = %s", listResponse.Code, listResponse.Body.String())
	}
	if !strings.Contains(listResponse.Body.String(), exchangeID.String()) {
		t.Fatalf("list does not contain exchange %s: %s", exchangeID, listResponse.Body.String())
	}
	if strings.Count(listResponse.Body.String(), `"position"`) != 3 {
		t.Fatalf("list must contain 3 participants: %s", listResponse.Body.String())
	}

	detailResponse := performRequest(
		service,
		http.MethodGet,
		"/exchanges/"+exchangeID.String(),
		authenticateAs(found[0].Participants[1].User.ID),
	)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("detail status = %d; body = %s", detailResponse.Code, detailResponse.Body.String())
	}
}

func cleanupIntegrationData(
	ctx context.Context,
	pool *pgxpool.Pool,
	users []uuid.UUID,
	items []uuid.UUID,
) {
	for _, userID := range users {
		_, _ = pool.Exec(ctx, `
			DELETE FROM chains
			WHERE id IN (
				SELECT chain_id
				FROM chain_participants
				WHERE user_id = $1
			)`, userID)
	}

	for _, itemID := range items {
		_, _ = pool.Exec(ctx, "DELETE FROM items WHERE id = $1", itemID)
	}

	for _, userID := range users {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	}
}
