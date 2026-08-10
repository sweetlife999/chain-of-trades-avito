//go:build integration

package handler

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
	userservice "github.com/sweetlife999/chain-of-trades-avito/internal/user/service"
)

func TestUserBlocksIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	users := []uuid.UUID{uuid.New(), uuid.New()}
	items := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	itemOwners := []uuid.UUID{users[0], users[1], users[0], users[1]}
	var exchanges []uuid.UUID
	t.Cleanup(func() {
		cleanupUserBlocksIntegration(context.Background(), pool, exchanges, items, users)
		pool.Close()
	})

	for index, userID := range users {
		_, err := pool.Exec(
			ctx,
			"INSERT INTO users (id, nickname, password_hash) VALUES ($1, $2, $3)",
			userID,
			"blocks-"+userID.String()[:8],
			"not-used-in-integration-test",
		)
		if err != nil {
			t.Fatalf("create user %d: %v", index, err)
		}
	}

	categories := []string{"books", "phones", "books", "phones"}
	for index, itemID := range items {
		_, err := pool.Exec(ctx, `
			INSERT INTO items (id, owner_id, category_id, title, photo_urls, created_at)
			VALUES (
				$1,
				$2,
				(SELECT id FROM categories WHERE slug = $3),
				$4,
				ARRAY['https://example.com/blocks.jpg'],
				'2000-01-01 00:00:00+00'::timestamptz + ($5 * interval '1 second')
			)`,
			itemID,
			itemOwners[index],
			categories[index],
			"Blocks integration item",
			index,
		)
		if err != nil {
			t.Fatalf("create item %d: %v", index, err)
		}

		wantedCategory := categories[(index+1)%len(categories)]
		_, err = pool.Exec(ctx, `
			INSERT INTO item_wants (item_id, category_id)
			VALUES ($1, (SELECT id FROM categories WHERE slug = $2))`,
			itemID,
			wantedCategory,
		)
		if err != nil {
			t.Fatalf("create item want %d: %v", index, err)
		}
	}

	exchangeRepository := exchangerepository.New(pool)
	exchangeService := exchangeservice.New(exchangeRepository)
	participants := []exchangemodel.Participant{
		{UserID: users[0], GivesItemID: items[0], ReceivesItemID: items[1], Position: 0},
		{UserID: users[1], GivesItemID: items[1], ReceivesItemID: items[0], Position: 1},
	}
	confirmedParticipants := []exchangemodel.Participant{
		{UserID: users[0], GivesItemID: items[2], ReceivesItemID: items[3], Position: 0},
		{UserID: users[1], GivesItemID: items[3], ReceivesItemID: items[2], Position: 1},
	}

	proposedID, err := exchangeRepository.SaveExchange(ctx, exchangemodel.Exchange{Participants: participants})
	if err != nil {
		t.Fatalf("create proposed exchange: %v", err)
	}
	exchanges = append(exchanges, proposedID)
	confirmedID, err := exchangeRepository.SaveExchange(ctx, exchangemodel.Exchange{Participants: confirmedParticipants})
	if err != nil {
		t.Fatalf("create confirmed exchange: %v", err)
	}
	exchanges = append(exchanges, confirmedID)
	if _, err := pool.Exec(ctx, "UPDATE chains SET status = 'confirmed' WHERE id = $1", confirmedID); err != nil {
		t.Fatalf("mark exchange confirmed: %v", err)
	}

	cycle, err := exchangeService.FindCycle(ctx, exchangemodel.Node{ItemID: items[0], OwnerID: users[0]})
	if err != nil || len(cycle) != 2 {
		t.Fatalf("cycle before block = %+v, error = %v; want two participants", cycle, err)
	}

	userService := userservice.New(userrepository.New(db.New(pool)))
	for attempt := 0; attempt < 2; attempt++ {
		response := performRequestWithAuth(
			userService,
			http.MethodPost,
			"/users/me/blocks/"+users[1].String(),
			"",
			authenticateAs(users[0]),
		)
		if response.Code != http.StatusNoContent {
			t.Fatalf("block attempt %d status = %d; body = %s", attempt+1, response.Code, response.Body.String())
		}
	}

	listResponse := performRequestWithAuth(
		userService,
		http.MethodGet,
		"/users/me/blocks",
		"",
		authenticateAs(users[0]),
	)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list blocks status = %d; body = %s", listResponse.Code, listResponse.Body.String())
	}

	var blockCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM user_blocks
		WHERE blocker_id = $1 AND blocked_id = $2`, users[0], users[1]).Scan(&blockCount); err != nil {
		t.Fatalf("count blocks: %v", err)
	}
	if blockCount != 1 {
		t.Fatalf("block count = %d, want 1", blockCount)
	}

	assertExchangeStatus(t, ctx, pool, proposedID, "cancelled")
	assertExchangeStatus(t, ctx, pool, confirmedID, "confirmed")
	var cancelReason string
	if err := pool.QueryRow(ctx, "SELECT cancel_reason FROM chains WHERE id = $1", proposedID).Scan(
		&cancelReason,
	); err != nil {
		t.Fatalf("read user-block cancellation reason: %v", err)
	}
	if cancelReason != "user_blocked" {
		t.Fatalf("user-block cancellation reason = %q, want user_blocked", cancelReason)
	}

	for index, scenario := range []struct {
		start         exchangemodel.Node
		forbiddenUser uuid.UUID
	}{
		{start: exchangemodel.Node{ItemID: items[0], OwnerID: users[0]}, forbiddenUser: users[1]},
		{start: exchangemodel.Node{ItemID: items[1], OwnerID: users[1]}, forbiddenUser: users[0]},
	} {
		cycle, err := exchangeService.FindCycle(ctx, scenario.start)
		if err != nil {
			t.Fatalf("find blocked cycle from side %d: %v", index, err)
		}
		for _, node := range cycle {
			if node.OwnerID == scenario.forbiddenUser {
				t.Fatalf(
					"blocked cycle from side %d contains forbidden user %s: %+v",
					index,
					scenario.forbiddenUser,
					cycle,
				)
			}
		}
	}

	unblockResponse := performRequestWithAuth(
		userService,
		http.MethodDelete,
		"/users/me/blocks/"+users[1].String(),
		"",
		authenticateAs(users[0]),
	)
	if unblockResponse.Code != http.StatusNoContent {
		t.Fatalf("unblock status = %d; body = %s", unblockResponse.Code, unblockResponse.Body.String())
	}

	cycle, err = exchangeService.FindCycle(ctx, exchangemodel.Node{ItemID: items[0], OwnerID: users[0]})
	if err != nil || len(cycle) != 2 {
		t.Fatalf("cycle after unblock = %+v, error = %v; want two participants", cycle, err)
	}

	selfResponse := performRequestWithAuth(
		userService,
		http.MethodPost,
		"/users/me/blocks/"+users[0].String(),
		"",
		authenticateAs(users[0]),
	)
	if selfResponse.Code != http.StatusBadRequest {
		t.Fatalf("self block status = %d, want %d", selfResponse.Code, http.StatusBadRequest)
	}

	missingResponse := performRequestWithAuth(
		userService,
		http.MethodPost,
		"/users/me/blocks/"+uuid.New().String(),
		"",
		authenticateAs(users[0]),
	)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing user block status = %d, want %d; body = %s", missingResponse.Code, http.StatusNotFound, missingResponse.Body.String())
	}
}

func assertExchangeStatus(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	exchangeID uuid.UUID,
	want string,
) {
	t.Helper()

	var status string
	if err := pool.QueryRow(ctx, "SELECT status::text FROM chains WHERE id = $1", exchangeID).Scan(&status); err != nil {
		t.Fatalf("read exchange %s status: %v", exchangeID, err)
	}
	if status != want {
		t.Fatalf("exchange %s status = %q, want %q", exchangeID, status, want)
	}
}

func cleanupUserBlocksIntegration(
	ctx context.Context,
	pool *pgxpool.Pool,
	exchangeIDs []uuid.UUID,
	itemIDs []uuid.UUID,
	userIDs []uuid.UUID,
) {
	_, _ = pool.Exec(ctx, `
		DELETE FROM broken_exchange_compositions
		WHERE source_chain_id = ANY($1::uuid[])
		   OR source_chain_id IN (
			SELECT participant.chain_id
			FROM chain_participants AS participant
			WHERE participant.user_id = ANY($2::uuid[])
		)`, exchangeIDs, userIDs)
	_, _ = pool.Exec(ctx, `
		DELETE FROM chains
		WHERE id = ANY($1::uuid[])
		   OR id IN (
			SELECT participant.chain_id
			FROM chain_participants AS participant
			WHERE participant.user_id = ANY($2::uuid[])
		)`, exchangeIDs, userIDs)
	_, _ = pool.Exec(ctx, "DELETE FROM items WHERE id = ANY($1::uuid[])", itemIDs)
	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = ANY($1::uuid[])", userIDs)
}
