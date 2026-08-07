//go:build integration

package handler

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
)

func TestExchangeRecoveryIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	users := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	items := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	t.Cleanup(func() {
		cleanupIntegrationData(context.Background(), pool, users, items)
		pool.Close()
	})

	for index, userID := range users {
		if _, err := pool.Exec(
			ctx,
			"INSERT INTO users (id, nickname, password_hash) VALUES ($1, $2, $3)",
			userID,
			"recovery-"+userID.String()[:8],
			"not-used-in-integration-test",
		); err != nil {
			t.Fatalf("create user %d: %v", index, err)
		}
	}

	categories := []string{"books", "hobby", "sports", "sports", "sports"}
	wants := []string{"hobby", "sports", "books", "books", "books"}
	for index, itemID := range items {
		if _, err := pool.Exec(ctx, `
			INSERT INTO items (id, owner_id, category_id, title, photo_urls, created_at)
			VALUES (
				$1,
				$2,
				(SELECT id FROM categories WHERE slug = $3),
				'Recovery integration item',
				ARRAY['https://example.com/recovery.jpg'],
				now() + ($4 * interval '1 second')
			)`,
			itemID,
			users[index],
			categories[index],
			index,
		); err != nil {
			t.Fatalf("create item %d: %v", index, err)
		}

		if _, err := pool.Exec(ctx, `
			INSERT INTO item_wants (item_id, category_id)
			VALUES ($1, (SELECT id FROM categories WHERE slug = $2))`,
			itemID,
			wants[index],
		); err != nil {
			t.Fatalf("create item wants %d: %v", index, err)
		}
	}

	repository := exchangerepository.New(pool)
	service := exchangeservice.New(repository)
	originalParticipants := []exchangemodel.Participant{
		{UserID: users[0], GivesItemID: items[0], ReceivesItemID: items[1], Position: 0},
		{UserID: users[1], GivesItemID: items[1], ReceivesItemID: items[2], Position: 1},
		{UserID: users[2], GivesItemID: items[2], ReceivesItemID: items[0], Position: 2},
	}
	originalID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: originalParticipants})
	if err != nil {
		t.Fatalf("create original exchange: %v", err)
	}

	if err := service.DeclineParticipation(ctx, originalID, users[2]); err != nil {
		t.Fatalf("decline proposed exchange: %v", err)
	}
	assertChainStatus(t, ctx, pool, originalID, "cancelled")

	var proposedAfterDecline int
	if err := pool.QueryRow(ctx, `
		SELECT count(DISTINCT chain.id)
		FROM chains AS chain
		JOIN chain_participants AS participant ON participant.chain_id = chain.id
		WHERE chain.status = 'proposed'
		  AND participant.gives_item_id = ANY($1::uuid[])`, items).Scan(&proposedAfterDecline); err != nil {
		t.Fatalf("count proposals after proposed decline: %v", err)
	}
	if proposedAfterDecline != 0 {
		t.Fatalf("proposals after proposed decline = %d, want 0", proposedAfterDecline)
	}

	confirmedParticipants := []exchangemodel.Participant{
		{UserID: users[0], GivesItemID: items[0], ReceivesItemID: items[1], Position: 0},
		{UserID: users[1], GivesItemID: items[1], ReceivesItemID: items[3], Position: 1},
		{UserID: users[3], GivesItemID: items[3], ReceivesItemID: items[0], Position: 2},
	}
	confirmedID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: confirmedParticipants})
	if err != nil {
		t.Fatalf("create exchange that will break: %v", err)
	}
	for _, userID := range []uuid.UUID{users[0], users[1], users[3]} {
		if err := service.ConfirmParticipation(ctx, confirmedID, userID); err != nil {
			t.Fatalf("confirm exchange for user %s: %v", userID, err)
		}
	}

	if err := service.DeclineParticipation(ctx, confirmedID, users[3]); err != nil {
		t.Fatalf("decline confirmed exchange: %v", err)
	}
	assertChainStatus(t, ctx, pool, confirmedID, "cancelled")

	var recoveredID uuid.UUID
	if err := pool.QueryRow(ctx, `
		SELECT chain.id
		FROM chains AS chain
		WHERE chain.status = 'proposed'
		  AND NOT EXISTS (
			SELECT 1
			FROM unnest($1::uuid[]) AS expected(item_id)
			WHERE NOT EXISTS (
				SELECT 1
				FROM chain_participants AS participant
				WHERE participant.chain_id = chain.id
				  AND participant.gives_item_id = expected.item_id
			)
		  )
		  AND (SELECT count(*) FROM chain_participants WHERE chain_id = chain.id) = cardinality($1::uuid[])
		LIMIT 1`,
		[]uuid.UUID{items[0], items[1], items[4]},
	).Scan(&recoveredID); err != nil {
		t.Fatalf("find recovered exchange with a new exact composition: %v", err)
	}

	for _, itemID := range []uuid.UUID{items[0], items[1], items[3]} {
		var status string
		if err := pool.QueryRow(ctx, "SELECT status FROM items WHERE id = $1", itemID).Scan(&status); err != nil {
			t.Fatalf("read released item %s: %v", itemID, err)
		}
		if status != "available" {
			t.Fatalf("item %s status = %q, want available", itemID, status)
		}
	}

	for index, userID := range users {
		var dealsBroken int
		if err := pool.QueryRow(ctx, "SELECT deals_broken FROM users WHERE id = $1", userID).Scan(&dealsBroken); err != nil {
			t.Fatalf("read user %d deals_broken: %v", index, err)
		}
		want := 0
		if userID == users[3] {
			want = 1
		}
		if dealsBroken != want {
			t.Fatalf("user %d deals_broken = %d, want %d", index, dealsBroken, want)
		}
	}
}

func assertChainStatus(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	exchangeID uuid.UUID,
	want string,
) {
	t.Helper()

	var status string
	if err := pool.QueryRow(ctx, "SELECT status FROM chains WHERE id = $1", exchangeID).Scan(&status); err != nil {
		t.Fatalf("read exchange %s status: %v", exchangeID, err)
	}
	if status != want {
		t.Fatalf("exchange %s status = %q, want %q", exchangeID, status, want)
	}
}
