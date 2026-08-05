//go:build integration

package handler

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
)

func TestExchangeDecisionsIntegration(t *testing.T) {
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

	for index, userID := range users {
		_, err := pool.Exec(
			ctx,
			"INSERT INTO users (id, nickname, password_hash) VALUES ($1, $2, $3)",
			userID,
			"decision-"+userID.String()[:8],
			"not-used-in-integration-test",
		)
		if err != nil {
			t.Fatalf("create user %d: %v", index, err)
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO items (id, owner_id, category_id, title, photo_urls)
			VALUES (
				$1,
				$2,
				(SELECT id FROM categories WHERE slug = 'books'),
				$3,
				ARRAY['https://example.com/decision.jpg']
			)`,
			items[index],
			userID,
			"Decision integration item",
		)
		if err != nil {
			t.Fatalf("create item %d: %v", index, err)
		}
	}

	participants := []exchangemodel.Participant{
		{UserID: users[0], GivesItemID: items[0], ReceivesItemID: items[1], Position: 0},
		{UserID: users[1], GivesItemID: items[1], ReceivesItemID: items[2], Position: 1},
		{UserID: users[2], GivesItemID: items[2], ReceivesItemID: items[0], Position: 2},
	}

	repository := exchangerepository.New(pool)
	service := exchangeservice.New(repository)
	firstExchangeID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: participants})
	if err != nil {
		t.Fatalf("create first exchange: %v", err)
	}
	secondExchangeID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: participants})
	if err != nil {
		t.Fatalf("create competing exchange: %v", err)
	}

	outsiderResponse := performRequest(
		service,
		http.MethodPost,
		"/exchanges/"+firstExchangeID.String()+"/confirm",
		authenticateAs(uuid.New()),
	)
	if outsiderResponse.Code != http.StatusForbidden {
		t.Fatalf("outsider confirm status = %d, want %d", outsiderResponse.Code, http.StatusForbidden)
	}

	for _, exchangeID := range []uuid.UUID{firstExchangeID, secondExchangeID} {
		for index := 0; index < len(participants)-1; index++ {
			if err := service.ConfirmParticipation(ctx, exchangeID, participants[index].UserID); err != nil {
				t.Fatalf("confirm exchange %s participant %d: %v", exchangeID, index, err)
			}
		}
	}

	type result struct {
		exchangeID uuid.UUID
		err        error
	}
	results := make(chan result, 2)
	var waitGroup sync.WaitGroup
	for _, exchangeID := range []uuid.UUID{firstExchangeID, secondExchangeID} {
		waitGroup.Add(1)
		go func(exchangeID uuid.UUID) {
			defer waitGroup.Done()
			results <- result{
				exchangeID: exchangeID,
				err: service.ConfirmParticipation(
					ctx,
					exchangeID,
					participants[len(participants)-1].UserID,
				),
			}
		}(exchangeID)
	}
	waitGroup.Wait()
	close(results)

	var confirmedExchangeID uuid.UUID
	var cancelledExchangeID uuid.UUID
	for result := range results {
		switch {
		case result.err == nil:
			confirmedExchangeID = result.exchangeID
		case errors.Is(result.err, exchangeservice.ErrConflict):
			cancelledExchangeID = result.exchangeID
		default:
			t.Fatalf("concurrent confirmation returned unexpected error: %v", result.err)
		}
	}
	if confirmedExchangeID == uuid.Nil || cancelledExchangeID == uuid.Nil {
		t.Fatalf(
			"concurrent result = confirmed %s, cancelled %s; want one of each",
			confirmedExchangeID,
			cancelledExchangeID,
		)
	}

	confirmed, err := service.GetForUser(ctx, confirmedExchangeID, users[0])
	if err != nil {
		t.Fatalf("get confirmed exchange: %v", err)
	}
	if confirmed.Status != "confirmed" {
		t.Fatalf("exchange status = %q, want confirmed", confirmed.Status)
	}
	for _, participant := range confirmed.Participants {
		if participant.Status != "accepted" || participant.GivesItem.Status != "reserved" {
			t.Fatalf(
				"confirmed participant status = %q, item status = %q; want accepted/reserved",
				participant.Status,
				participant.GivesItem.Status,
			)
		}
	}

	repeatResponse := performRequest(
		service,
		http.MethodPost,
		"/exchanges/"+confirmedExchangeID.String()+"/confirm",
		authenticateAs(users[0]),
	)
	if repeatResponse.Code != http.StatusConflict {
		t.Fatalf("repeat confirm status = %d, want %d", repeatResponse.Code, http.StatusConflict)
	}

	cancelled, err := service.GetForUser(ctx, cancelledExchangeID, users[0])
	if err != nil {
		t.Fatalf("get cancelled exchange: %v", err)
	}
	if cancelled.Status != "cancelled" {
		t.Fatalf("competing exchange status = %q, want cancelled", cancelled.Status)
	}
	if cancelled.ClosedAt == nil {
		t.Fatal("competing exchange closed_at is nil")
	}
	for _, participant := range cancelled.Participants {
		if participant.GivesItem.Status != "reserved" {
			t.Fatalf(
				"cancelled competing exchange released an item from confirmed exchange: status = %q",
				participant.GivesItem.Status,
			)
		}
	}
}
