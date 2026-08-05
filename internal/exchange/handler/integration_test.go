//go:build integration

package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
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
	defer pool.Close()

	users := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	items := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	t.Cleanup(func() {
		cleanupIntegrationData(context.Background(), pool, users, items)
	})

	categorySlugs := []string{"books", "hobby", "sports"}
	categories := make([]int16, len(categorySlugs))
	for index, slug := range categorySlugs {
		if err := pool.QueryRow(ctx, "SELECT id FROM categories WHERE slug = $1", slug).
			Scan(&categories[index]); err != nil {
			t.Fatalf("get category %q: %v", slug, err)
		}
	}

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

	hasPhotoURLs := false
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'items'
			  AND column_name = 'photo_urls'
		)`).Scan(&hasPhotoURLs); err != nil {
		t.Fatalf("inspect item photo schema: %v", err)
	}

	for index, itemID := range items {
		if err := insertIntegrationItem(
			ctx,
			pool,
			hasPhotoURLs,
			itemID,
			users[index],
			categories[index],
			fmt.Sprintf("Integration item %d", index+1),
		); err != nil {
			t.Fatalf("create item %d: %v", index, err)
		}

		wantedCategory := categories[(index+1)%len(categories)]
		if _, err := pool.Exec(
			ctx,
			"INSERT INTO item_wants (item_id, category_id) VALUES ($1, $2)",
			itemID,
			wantedCategory,
		); err != nil {
			t.Fatalf("create item want %d: %v", index, err)
		}
	}

	repository := exchangerepository.New(pool)
	service := exchangeservice.New(repository)
	result, err := service.FindAndSave(ctx, exchangemodel.Node{
		ItemID:  items[0],
		OwnerID: users[0],
	})
	if err != nil {
		t.Fatalf("find and save exchange: %v", err)
	}
	if !result.Found {
		t.Fatal("exchange was not found")
	}

	listResponse := performRequest(service, http.MethodGet, "/exchanges", authenticateAs(users[0]))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d; body = %s", listResponse.Code, listResponse.Body.String())
	}
	if !strings.Contains(listResponse.Body.String(), result.ExchangeID.String()) {
		t.Fatalf("list does not contain exchange %s: %s", result.ExchangeID, listResponse.Body.String())
	}
	if strings.Count(listResponse.Body.String(), `"position"`) != 3 {
		t.Fatalf("list must contain 3 participants: %s", listResponse.Body.String())
	}

	detailResponse := performRequest(
		service,
		http.MethodGet,
		"/exchanges/"+result.ExchangeID.String(),
		authenticateAs(users[1]),
	)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("detail status = %d; body = %s", detailResponse.Code, detailResponse.Body.String())
	}
}

func insertIntegrationItem(
	ctx context.Context,
	pool *pgxpool.Pool,
	hasPhotoURLs bool,
	itemID uuid.UUID,
	ownerID uuid.UUID,
	categoryID int16,
	title string,
) error {
	if hasPhotoURLs {
		_, err := pool.Exec(ctx, `
			INSERT INTO items (id, owner_id, category_id, title, photo_urls)
			VALUES ($1, $2, $3, $4, ARRAY[$5]::text[])`,
			itemID,
			ownerID,
			categoryID,
			title,
			"https://example.com/integration.jpg",
		)
		return err
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO items (id, owner_id, category_id, title)
		VALUES ($1, $2, $3, $4)`,
		itemID,
		ownerID,
		categoryID,
		title,
	)
	return err
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
