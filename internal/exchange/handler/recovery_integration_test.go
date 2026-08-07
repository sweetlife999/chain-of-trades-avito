//go:build integration

package handler

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	users, items := seedExchangeGraph(
		t,
		ctx,
		pool,
		"recovery",
		[]string{"books", "hobby", "sports", "sports", "sports"},
		[]string{"hobby", "sports", "books", "books", "books"},
	)

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

	// Отказ от предложения тоже запускает перепоиск: ребро 2 -> 0 вырезано, поэтому DFS
	// собирает состав с другой третьей вещью, а не переподставляет отказанный.
	restored := proposedExchangesWithItems(t, ctx, pool, items[0], items[1], items[3])
	if len(restored) != 1 {
		t.Fatalf("proposals after proposed decline = %d, want exactly one with a new third item", len(restored))
	}
	if refused := proposedExchangesWithItems(t, ctx, pool, items[0], items[1], items[2]); len(refused) != 0 {
		t.Fatalf("re-proposed the refused composition %d times, want 0", len(refused))
	}

	confirmedID := restored[0]
	for _, userID := range []uuid.UUID{users[0], users[1], users[3]} {
		if err := service.ConfirmParticipation(ctx, confirmedID, userID); err != nil {
			t.Fatalf("confirm exchange for user %s: %v", userID, err)
		}
	}

	if err := service.DeclineParticipation(ctx, confirmedID, users[3]); err != nil {
		t.Fatalf("decline confirmed exchange: %v", err)
	}
	assertChainStatus(t, ctx, pool, confirmedID, "cancelled")

	if found := proposedExchangesWithItems(t, ctx, pool, items[0], items[1], items[4]); len(found) != 1 {
		t.Fatalf("recovered exchanges with a new exact composition = %d, want 1", len(found))
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

// TestExchangeSupersededRecoveryIntegration — сценарий issue #55. Две цепи с общим
// участником: первая подтверждается и вытесняет вторую, потом срывается сама. Вторая
// цепь никто не отклонял, поэтому она обязана собраться заново.
func TestExchangeSupersededRecoveryIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	// Граф держит два цикла с общей вещью 2: 0 -> 1 -> 2 -> 0 и 2 -> 3 -> 4 -> 2.
	users, items := seedExchangeGraph(
		t,
		ctx,
		pool,
		"superseded",
		[]string{"books", "hobby", "sports", "books", "tools"},
		[]string{"hobby", "sports", "books", "tools", "sports"},
	)

	repository := exchangerepository.New(pool)
	service := exchangeservice.New(repository)

	winnerID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: []exchangemodel.Participant{
		{UserID: users[0], GivesItemID: items[0], ReceivesItemID: items[1], Position: 0},
		{UserID: users[1], GivesItemID: items[1], ReceivesItemID: items[2], Position: 1},
		{UserID: users[2], GivesItemID: items[2], ReceivesItemID: items[0], Position: 2},
	}})
	if err != nil {
		t.Fatalf("create exchange that will win: %v", err)
	}

	supersededID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: []exchangemodel.Participant{
		{UserID: users[2], GivesItemID: items[2], ReceivesItemID: items[3], Position: 0},
		{UserID: users[3], GivesItemID: items[3], ReceivesItemID: items[4], Position: 1},
		{UserID: users[4], GivesItemID: items[4], ReceivesItemID: items[2], Position: 2},
	}})
	if err != nil {
		t.Fatalf("create exchange that will be superseded: %v", err)
	}

	for _, userID := range []uuid.UUID{users[0], users[1], users[2]} {
		if err := service.ConfirmParticipation(ctx, winnerID, userID); err != nil {
			t.Fatalf("confirm winning exchange for user %s: %v", userID, err)
		}
	}
	assertChainStatus(t, ctx, pool, winnerID, "confirmed")
	// Вторую цепь закрыло чужое подтверждение, а не отказ её участника.
	assertChainStatus(t, ctx, pool, supersededID, "cancelled")

	if err := service.DeclineParticipation(ctx, winnerID, users[0]); err != nil {
		t.Fatalf("break the winning exchange: %v", err)
	}
	assertChainStatus(t, ctx, pool, winnerID, "cancelled")

	if found := proposedExchangesWithItems(t, ctx, pool, items[2], items[3], items[4]); len(found) != 1 {
		t.Fatalf("restored superseded exchanges = %d, want 1", len(found))
	}
	// Сорванный состав перепоиск переподставлять не должен: от него только что ушли.
	if found := proposedExchangesWithItems(t, ctx, pool, items[0], items[1], items[2]); len(found) != 0 {
		t.Fatalf("re-proposed the broken exchange %d times, want 0", len(found))
	}
}

// seedExchangeGraph заводит по пользователю на каждую категорию из categories и по
// объявлению на каждого: категория объявления берётся из categories, желаемая — из wants
// того же индекса. Объявления создаются с разбегом по created_at, чтобы DFS при одинаковых
// данных обходил их в одном и том же порядке. Данные удаляются после теста.
func seedExchangeGraph(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	prefix string,
	categories []string,
	wants []string,
) ([]uuid.UUID, []uuid.UUID) {
	t.Helper()

	users := make([]uuid.UUID, len(categories))
	items := make([]uuid.UUID, len(categories))
	for index := range categories {
		users[index] = uuid.New()
		items[index] = uuid.New()
	}
	t.Cleanup(func() {
		cleanupIntegrationData(context.Background(), pool, users, items)
		pool.Close()
	})

	for index, userID := range users {
		if _, err := pool.Exec(
			ctx,
			"INSERT INTO users (id, nickname, password_hash) VALUES ($1, $2, $3)",
			userID,
			prefix+"-"+userID.String()[:8],
			"not-used-in-integration-test",
		); err != nil {
			t.Fatalf("create user %d: %v", index, err)
		}
	}

	for index, itemID := range items {
		if _, err := pool.Exec(ctx, `
			INSERT INTO items (id, owner_id, category_id, title, photo_urls, created_at)
			VALUES (
				$1,
				$2,
				(SELECT id FROM categories WHERE slug = $3),
				$4,
				ARRAY['https://example.com/integration.jpg'],
				now() + ($5 * interval '1 second')
			)`,
			itemID,
			users[index],
			categories[index],
			prefix+" integration item",
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

	return users, items
}

// proposedExchangesWithItems возвращает открытые предложения, состав которых совпадает с
// перечисленными объявлениями ровно, без учёта порядка обхода.
func proposedExchangesWithItems(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	items ...uuid.UUID,
) []uuid.UUID {
	t.Helper()

	rows, err := pool.Query(ctx, `
		SELECT chain.id
		FROM chains AS chain
		WHERE chain.status = 'proposed'
		  AND (
			SELECT array_agg(participant.gives_item_id ORDER BY participant.gives_item_id)
			FROM chain_participants AS participant
			WHERE participant.chain_id = chain.id
		  ) = (SELECT array_agg(item ORDER BY item) FROM unnest($1::uuid[]) AS item)`,
		items,
	)
	if err != nil {
		t.Fatalf("query proposed exchanges with items %v: %v", items, err)
	}

	found, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		t.Fatalf("read proposed exchanges with items %v: %v", items, err)
	}

	return found
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
