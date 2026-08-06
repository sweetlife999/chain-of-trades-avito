//go:build integration

package handler

import (
	"context"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
)

// TestExchangeMessagesIntegration проходит тред обмена целиком на живой БД: переписка
// участников, события сделки в той же ленте, счётчик непрочитанного и запрет писать
// в закрытый обмен.
func TestExchangeMessagesIntegration(t *testing.T) {
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
	items := []uuid.UUID{uuid.New(), uuid.New()}
	t.Cleanup(func() {
		cleanupIntegrationData(context.Background(), pool, users, items)
		pool.Close()
	})

	for index, userID := range users {
		_, err := pool.Exec(
			ctx,
			"INSERT INTO users (id, nickname, password_hash) VALUES ($1, $2, $3)",
			userID,
			"messages-"+userID.String()[:8],
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
				ARRAY['https://example.com/messages.jpg']
			)`,
			items[index],
			userID,
			"Messages integration item",
		)
		if err != nil {
			t.Fatalf("create item %d: %v", index, err)
		}
	}

	alice, bob := users[0], users[1]
	participants := []exchangemodel.Participant{
		{UserID: alice, GivesItemID: items[0], ReceivesItemID: items[1], Position: 0},
		{UserID: bob, GivesItemID: items[1], ReceivesItemID: items[0], Position: 1},
	}

	repository := exchangerepository.New(pool)
	service := exchangeservice.New(repository)

	exchangeID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: participants})
	if err != nil {
		t.Fatalf("create exchange: %v", err)
	}
	// Второй обмен на тех же вещах: подтверждение первого обязано закрыть его и объяснить
	// участникам, почему он закрылся.
	competingID, err := repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: participants})
	if err != nil {
		t.Fatalf("create competing exchange: %v", err)
	}

	messagesPath := "/exchanges/" + exchangeID.String() + "/messages"

	// Посторонний не видит тред и не может в него написать.
	outsider := authenticateAs(uuid.New())
	if response := performRequest(service, http.MethodGet, messagesPath, outsider); response.Code != http.StatusForbidden {
		t.Fatalf("outsider read status = %d, want %d", response.Code, http.StatusForbidden)
	}
	response := performRequestWithBody(
		service,
		http.MethodPost,
		messagesPath,
		strings.NewReader(`{"body":"пустите меня"}`),
		outsider,
	)
	if response.Code != http.StatusForbidden {
		t.Fatalf("outsider write status = %d, want %d", response.Code, http.StatusForbidden)
	}

	// Пустое сообщение до базы не доходит.
	response = performRequestWithBody(
		service,
		http.MethodPost,
		messagesPath,
		strings.NewReader(`{"body":"   "}`),
		authenticateAs(alice),
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("empty body status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	response = performRequestWithBody(
		service,
		http.MethodPost,
		messagesPath,
		strings.NewReader(`{"body":"могу привезти в субботу"}`),
		authenticateAs(alice),
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("post message status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}

	if unread := unreadCount(t, service, bob, exchangeID); unread != 1 {
		t.Fatalf("unread count for the recipient = %d, want 1", unread)
	}
	if unread := unreadCount(t, service, alice, exchangeID); unread != 0 {
		t.Fatalf("unread count for the author = %d, want 0: own messages are not unread", unread)
	}

	messages, err := service.ListMessages(ctx, exchangeID, bob)
	if err != nil {
		t.Fatalf("read thread: %v", err)
	}
	if len(messages) != 1 || messages[0].Author == nil || messages[0].Author.ID != alice {
		t.Fatalf("thread = %+v, want one message from the first participant", messages)
	}

	// Чтение треда и есть отметка о прочтении.
	if unread := unreadCount(t, service, bob, exchangeID); unread != 0 {
		t.Fatalf("unread count after reading = %d, want 0", unread)
	}

	for _, userID := range []uuid.UUID{alice, bob} {
		if err := service.ConfirmParticipation(ctx, exchangeID, userID); err != nil {
			t.Fatalf("confirm participation: %v", err)
		}
	}

	// События сделки лежат в той же ленте и в порядке, в котором произошли.
	assertThreadKinds(t, service, exchangeID, alice,
		"text",
		"participant_accepted",
		"participant_accepted",
		"exchange_confirmed",
	)

	// Обмен на тех же вещах закрылся сам, и в его треде написано почему.
	assertThreadKinds(t, service, competingID, alice, "exchange_superseded")

	response = performRequestWithBody(
		service,
		http.MethodPost,
		"/exchanges/"+competingID.String()+"/messages",
		strings.NewReader(`{"body":"а что случилось?"}`),
		authenticateAs(alice),
	)
	if response.Code != http.StatusConflict {
		t.Fatalf("write into a closed exchange status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func unreadCount(
	t *testing.T,
	service *exchangeservice.Service,
	userID uuid.UUID,
	exchangeID uuid.UUID,
) int64 {
	t.Helper()

	exchanges, err := service.ListForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("list exchanges: %v", err)
	}

	for _, exchange := range exchanges {
		if exchange.ID == exchangeID {
			return exchange.UnreadCount
		}
	}

	t.Fatalf("exchange %s is missing from the user's list", exchangeID)
	return 0
}

func assertThreadKinds(
	t *testing.T,
	service *exchangeservice.Service,
	exchangeID uuid.UUID,
	userID uuid.UUID,
	want ...string,
) {
	t.Helper()

	messages, err := service.ListMessages(context.Background(), exchangeID, userID)
	if err != nil {
		t.Fatalf("read thread: %v", err)
	}

	kinds := make([]string, len(messages))
	for index, message := range messages {
		kinds[index] = message.Kind
	}

	if !slices.Equal(kinds, want) {
		t.Fatalf("thread = %v, want %v", kinds, want)
	}
}
