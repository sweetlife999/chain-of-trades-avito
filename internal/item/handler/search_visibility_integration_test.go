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

func TestItemSearchVisibilityIntegration(t *testing.T) {
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
	items := make([]uuid.UUID, 0, len(users))
	t.Cleanup(func() {
		cleanupSearchVisibilityData(context.Background(), pool, users, items)
		pool.Close()
	})

	for index, userID := range users {
		if _, err := pool.Exec(ctx, `
			INSERT INTO users (id, nickname, password_hash)
			VALUES ($1, $2, 'not-used-in-integration-test')`,
			userID, "visibility-"+userID.String()[:8]); err != nil {
			t.Fatalf("create user %d: %v", index, err)
		}
	}

	exchanges := exchangeservice.New(exchangerepository.New(pool))
	itemsService := itemservice.New(itemrepository.New(pool), exchanges)
	categories := []string{"books", "hobby", "sports"}
	wants := []string{"hobby", "sports", "books"}
	for index := range users {
		created, err := itemsService.Create(ctx, itemservice.CreateInput{
			OwnerID:   users[index],
			Category:  categories[index],
			Title:     "Search visibility integration item",
			PhotoURLs: []string{"https://example.com/integration.jpg"},
			Wants:     []string{wants[index]},
		})
		if err != nil {
			t.Fatalf("create item %d: %v", index, err)
		}
		items = append(items, created.ID)
	}

	found, err := exchanges.ListForUser(ctx, users[0])
	if err != nil {
		t.Fatalf("list initial exchanges: %v", err)
	}
	if len(found) != 1 || found[0].Status != "proposed" {
		t.Fatalf("initial exchanges = %+v, want one proposed", found)
	}
	originalExchangeID := found[0].ID

	withdrawResponse := performRequest(
		itemsService,
		http.MethodDelete,
		"/items/"+items[0].String()+"/search",
		"",
		authenticateAs(users[0]),
	)
	if withdrawResponse.Code != http.StatusOK ||
		!strings.Contains(withdrawResponse.Body.String(), `"status":"withdrawn"`) {
		t.Fatalf("withdraw response = %d %s", withdrawResponse.Code, withdrawResponse.Body.String())
	}

	assertIntegrationStatus(t, ctx, pool, "items", items[0], "withdrawn")
	assertIntegrationStatus(t, ctx, pool, "chains", originalExchangeID, "cancelled")
	var cancelReason string
	if err := pool.QueryRow(
		ctx,
		"SELECT cancel_reason FROM chains WHERE id = $1",
		originalExchangeID,
	).Scan(&cancelReason); err != nil {
		t.Fatalf("read item-withdrawal cancellation reason: %v", err)
	}
	if cancelReason != "item_withdrawn" {
		t.Fatalf("item-withdrawal cancellation reason = %q, want item_withdrawn", cancelReason)
	}
	var withdrawalEvents int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM chain_messages
		WHERE chain_id = $1 AND kind = 'exchange_item_withdrawn'`, originalExchangeID).Scan(&withdrawalEvents); err != nil {
		t.Fatalf("count withdrawal events: %v", err)
	}
	if withdrawalEvents != 1 {
		t.Fatalf("withdrawal events = %d, want 1", withdrawalEvents)
	}

	publishResponse := performRequest(
		itemsService,
		http.MethodPut,
		"/items/"+items[0].String()+"/search",
		"",
		authenticateAs(users[0]),
	)
	if publishResponse.Code != http.StatusOK ||
		!strings.Contains(publishResponse.Body.String(), `"status":"available"`) {
		t.Fatalf("publish response = %d %s", publishResponse.Code, publishResponse.Body.String())
	}

	refound, err := exchanges.ListForUser(ctx, users[0])
	if err != nil {
		t.Fatalf("list exchanges after publishing: %v", err)
	}
	var replacementID uuid.UUID
	for _, exchange := range refound {
		if exchange.Status == "proposed" {
			replacementID = exchange.ID
		}
	}
	if replacementID == uuid.Nil || replacementID == originalExchangeID {
		t.Fatalf("replacement exchange = %s, original = %s", replacementID, originalExchangeID)
	}

	for _, userID := range users {
		if err := exchanges.ConfirmParticipation(ctx, replacementID, userID); err != nil {
			t.Fatalf("confirm replacement for user %s: %v", userID, err)
		}
	}
	assertIntegrationStatus(t, ctx, pool, "chains", replacementID, "confirmed")

	conflictResponse := performRequest(
		itemsService,
		http.MethodDelete,
		"/items/"+items[0].String()+"/search",
		"",
		authenticateAs(users[0]),
	)
	if conflictResponse.Code != http.StatusConflict {
		t.Fatalf("withdraw reserved response = %d %s, want 409", conflictResponse.Code, conflictResponse.Body.String())
	}
	assertIntegrationStatus(t, ctx, pool, "items", items[0], "reserved")
}

func assertIntegrationStatus(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	table string,
	id uuid.UUID,
	want string,
) {
	t.Helper()

	var status string
	query := "SELECT status FROM " + table + " WHERE id = $1" // #nosec G202 -- table is a test constant.
	if err := pool.QueryRow(ctx, query, id).Scan(&status); err != nil {
		t.Fatalf("read %s %s status: %v", table, id, err)
	}
	if status != want {
		t.Fatalf("%s %s status = %q, want %q", table, id, status, want)
	}
}

func cleanupSearchVisibilityData(
	ctx context.Context,
	pool *pgxpool.Pool,
	users []uuid.UUID,
	items []uuid.UUID,
) {
	_, _ = pool.Exec(ctx, `
		DELETE FROM broken_exchange_compositions
		WHERE source_chain_id IN (
			SELECT chain_id FROM chain_participants WHERE user_id = ANY($1::uuid[])
		)`, users)
	for _, userID := range users {
		_, _ = pool.Exec(ctx, `
			DELETE FROM chains
			WHERE id IN (
				SELECT chain_id FROM chain_participants WHERE user_id = $1
			)`, userID)
	}
	for _, itemID := range items {
		_, _ = pool.Exec(ctx, "DELETE FROM items WHERE id = $1", itemID)
	}
	for _, userID := range users {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	}
}
