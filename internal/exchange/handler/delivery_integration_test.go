//go:build integration

package handler

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
)

// TestExchangeDeliveryIntegration гоняет узел доставки на живой БД: вещи уезжают в пункт,
// последняя сдача уводит обмен в delivering, «Товар получен» до выдачи не проходит, а после
// завершения пункт с вещей снимается. Перевод delivering -> delivered выполняет администратор.
func TestExchangeDeliveryIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	// Две независимые пары: первая проходит обычный путь, вторая сдаёт вещи до сборки обмена.
	users, items := seedExchangeGraph(
		t,
		ctx,
		pool,
		"delivery",
		[]string{"books", "hobby", "sports", "tools"},
		[]string{"hobby", "books", "tools", "sports"},
	)

	repository := exchangerepository.New(pool)
	service := exchangeservice.New(repository)
	pointID := createPickupPoint(ctx, t, pool)

	exchangeID := saveTwoPartyExchange(t, ctx, repository, users[0], users[1], items[0], items[1])

	for index, userID := range users[:2] {
		if err := service.ConfirmParticipation(ctx, exchangeID, userID); err != nil {
			t.Fatalf("confirm participant %d: %v", index, err)
		}
	}
	assertChainStatus(t, ctx, pool, exchangeID, "confirmed")

	// Первая сдача обмен не двигает: вторую вещь ещё ждут.
	if err := service.RecordItemPickup(ctx, items[0], users[0], pointID); err != nil {
		t.Fatalf("record first item pickup: %v", err)
	}
	assertChainStatus(t, ctx, pool, exchangeID, "confirmed")
	assertThreadTail(t, ctx, repository, exchangeID, "participant_delivered_item")

	if err := service.RecordItemPickup(ctx, items[1], users[1], pointID); err != nil {
		t.Fatalf("record second item pickup: %v", err)
	}
	assertChainStatus(t, ctx, pool, exchangeID, "delivering")
	assertThreadTail(t, ctx, repository, exchangeID, "participant_delivered_item", "exchange_delivering")

	// Пункты вещи ещё не выдали, значит подтверждать нечего.
	err = service.CompleteParticipation(ctx, exchangeID, users[0])
	if !errors.Is(err, exchangeservice.ErrConflict) {
		t.Fatalf("complete before delivery error = %v, want ErrConflict", err)
	}

	if err := service.MarkDeliveredByAdmin(ctx, exchangeID, users[0]); err != nil {
		t.Fatalf("mark exchange delivered: %v", err)
	}
	assertChainStatus(t, ctx, pool, exchangeID, "delivered")
	assertThreadTail(t, ctx, repository, exchangeID, "exchange_delivering", "exchange_delivered")

	for index, userID := range users[:2] {
		if err := service.CompleteParticipation(ctx, exchangeID, userID); err != nil {
			t.Fatalf("complete participant %d: %v", index, err)
		}
	}
	assertChainStatus(t, ctx, pool, exchangeID, "completed")

	// Завершение открывает оценку: партнёром назначен тот, чья вещь пришла, срок отсчитан
	// от закрытия обмена, а самой оценки ещё нет. Сама ручка живёт в internal/rating —
	// здесь проверяется только то, что обмен отдаёт участнику эти данные.
	completedExchange, err := service.GetForUser(ctx, exchangeID, users[0])
	if err != nil {
		t.Fatalf("get completed exchange: %v", err)
	}
	if completedExchange.Rating == nil {
		t.Fatal("завершённый обмен не отдал данные для оценки")
	}
	if completedExchange.Rating.RatedUserID != users[1] {
		t.Fatalf("оценить предлагают %s, а вещь пришла от %s",
			completedExchange.Rating.RatedUserID, users[1])
	}
	if completedExchange.Rating.Score != nil {
		t.Fatalf("оценка появилась сама: %d", *completedExchange.Rating.Score)
	}
	if !completedExchange.Rating.RateUntil.After(time.Now()) {
		t.Fatalf("срок оценки уже истёк: %s", completedExchange.Rating.RateUntil)
	}

	for index, itemID := range items[:2] {
		var status string
		var storedPoint *uuid.UUID
		err := pool.QueryRow(
			ctx,
			"SELECT status, pickup_point_id FROM items WHERE id = $1",
			itemID,
		).Scan(&status, &storedPoint)
		if err != nil {
			t.Fatalf("read item %d after completion: %v", index, err)
		}
		if status != "traded" {
			t.Fatalf("item %d status = %q, want traded", index, status)
		}
		// Вещь уехала к новому владельцу: адрес хранения больше ничего не описывает и
		// зря держал бы пункт от удаления.
		if storedPoint != nil {
			t.Fatalf("item %d still points at pickup point %s", index, *storedPoint)
		}
	}

	assertPreDeliveredExchangeGoesStraightToDelivering(t, ctx, pool, repository, service, users, items, pointID)
}

// Вещи можно отнести в пункт заранее. Тогда к моменту сборки обмена сдавать уже нечего, и
// подтверждение участия — единственный момент, когда цепочка может уйти в доставку.
func assertPreDeliveredExchangeGoesStraightToDelivering(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	repository *exchangerepository.Repository,
	service *exchangeservice.Service,
	users []uuid.UUID,
	items []uuid.UUID,
	pointID uuid.UUID,
) {
	t.Helper()

	for index := 2; index < 4; index++ {
		if err := service.RecordItemPickup(ctx, items[index], users[index], pointID); err != nil {
			t.Fatalf("record pickup before exchange for item %d: %v", index, err)
		}
	}

	exchangeID := saveTwoPartyExchange(t, ctx, repository, users[2], users[3], items[2], items[3])

	for index, userID := range users[2:4] {
		if err := service.ConfirmParticipation(ctx, exchangeID, userID); err != nil {
			t.Fatalf("confirm pre-delivered participant %d: %v", index, err)
		}
	}

	assertChainStatus(t, ctx, pool, exchangeID, "delivering")
	assertThreadTail(t, ctx, repository, exchangeID, "exchange_confirmed", "exchange_delivering")
}

func saveTwoPartyExchange(
	t *testing.T,
	ctx context.Context,
	repository *exchangerepository.Repository,
	firstUser uuid.UUID,
	secondUser uuid.UUID,
	firstItem uuid.UUID,
	secondItem uuid.UUID,
) uuid.UUID {
	t.Helper()

	exchangeID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: []exchangemodel.Participant{
		{UserID: firstUser, GivesItemID: firstItem, ReceivesItemID: secondItem, Position: 0},
		{UserID: secondUser, GivesItemID: secondItem, ReceivesItemID: firstItem, Position: 1},
	}})
	if err != nil {
		t.Fatalf("save exchange: %v", err)
	}

	return exchangeID
}

// assertThreadTail сверяет последние события треда: он растёт, и проверять его целиком
// значило бы переписывать тест на каждое новое событие сделки.
func assertThreadTail(
	t *testing.T,
	ctx context.Context,
	repository *exchangerepository.Repository,
	exchangeID uuid.UUID,
	want ...string,
) {
	t.Helper()

	messages, err := repository.ListMessages(ctx, exchangeID)
	if err != nil {
		t.Fatalf("list exchange thread: %v", err)
	}
	if len(messages) < len(want) {
		t.Fatalf("thread has %d messages, want at least %d", len(messages), len(want))
	}

	tail := messages[len(messages)-len(want):]
	for index, message := range tail {
		if message.Kind != want[index] {
			kinds := make([]string, len(messages))
			for position, recorded := range messages {
				kinds[position] = recorded.Kind
			}
			t.Fatalf("thread tail = %v, want it to end with %v", kinds, want)
		}
	}
}
