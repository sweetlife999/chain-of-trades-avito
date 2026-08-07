//go:build integration

package handler

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
)

// TestExchangeRefusalIntegration проверяет, что отказ запоминается как вырезанное ребро,
// а не как забаненный состав: отказавшемуся не переподставляют ту же вещь в цепочке с
// другим третьим участником, но остальные рёбра отказанного цикла остаются живыми.
func TestExchangeRefusalIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	// Граф держит отказанный цикл 0 -> 1 -> 2 -> 0, его вариацию 0 -> 1 -> 3 -> 0 с той же
	// вещью 1 и цикл 1 -> 2 -> 4 -> 1, который отказавшегося не касается вовсе.
	users, items := seedExchangeGraph(
		t,
		ctx,
		pool,
		"refusal",
		[]string{"books", "hobby", "sports", "sports", "books"},
		[]string{"hobby", "sports", "books", "books", "hobby"},
	)

	repository := exchangerepository.New(pool)
	service := exchangeservice.New(repository)

	refusedID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: []exchangemodel.Participant{
		{UserID: users[0], GivesItemID: items[0], ReceivesItemID: items[1], Position: 0},
		{UserID: users[1], GivesItemID: items[1], ReceivesItemID: items[2], Position: 1},
		{UserID: users[2], GivesItemID: items[2], ReceivesItemID: items[0], Position: 2},
	}})
	if err != nil {
		t.Fatalf("create exchange that will be refused: %v", err)
	}

	if err := service.DeclineParticipation(ctx, refusedID, users[0]); err != nil {
		t.Fatalf("decline proposed exchange: %v", err)
	}
	assertChainStatus(t, ctx, pool, refusedID, "cancelled")

	var refusedItems int
	if err := pool.QueryRow(
		ctx,
		"SELECT count(*) FROM item_refusals WHERE user_id = $1 AND item_id = $2",
		users[0],
		items[1],
	).Scan(&refusedItems); err != nil {
		t.Fatalf("read recorded refusal: %v", err)
	}
	if refusedItems != 1 {
		t.Fatalf("recorded refusals for the item the user turned down = %d, want 1", refusedItems)
	}

	// Перепоиск после отказа не предлагает вариацию с той же вещью 1: выучен сам отказ,
	// а не состав, в котором он прозвучал.
	if found := proposedExchangesWithItems(t, ctx, pool, items[0], items[1], items[3]); len(found) != 0 {
		t.Fatalf("proposed the refused item in another composition %d times, want 0", len(found))
	}

	// А ребро 1 -> 2 живо: ни владелец 1, ни владелец 2 ни от чего не отказывались,
	// поэтому цикл без отказавшегося собирается тем же перепоиском.
	if found := proposedExchangesWithItems(t, ctx, pool, items[1], items[2], items[4]); len(found) != 1 {
		t.Fatalf("exchanges with the surviving edge = %d, want 1", len(found))
	}
}
