//go:build integration

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	notificationdto "github.com/sweetlife999/chain-of-trades-avito/internal/notification/dto"
	notificationrepository "github.com/sweetlife999/chain-of-trades-avito/internal/notification/repository"
	notificationservice "github.com/sweetlife999/chain-of-trades-avito/internal/notification/service"
)

// TestNotificationsIntegration проверяет полный пользовательский поток на живой БД:
// предложение, чужое сообщение, системное событие, изоляцию пользователей и обе отметки
// прочтения. Именно триггеры связывают события приложения с таблицей notifications.
func TestNotificationsIntegration(t *testing.T) {
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
	items := []uuid.UUID{uuid.New(), uuid.New()}
	exchangeID := uuid.New()
	t.Cleanup(func() {
		cleanup := context.Background()
		if _, err := pool.Exec(cleanup, "DELETE FROM chains WHERE id = $1", exchangeID); err != nil {
			t.Errorf("cleanup chain: %v", err)
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM items WHERE id = ANY($1)", items); err != nil {
			t.Errorf("cleanup items: %v", err)
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE id = ANY($1)", users); err != nil {
			t.Errorf("cleanup users: %v", err)
		}
		pool.Close()
	})

	for index, userID := range users {
		if _, err := pool.Exec(
			ctx,
			"INSERT INTO users (id, nickname, password_hash) VALUES ($1, $2, $3)",
			userID,
			"notify-"+userID.String()[:8],
			"not-used-in-integration-test",
		); err != nil {
			t.Fatalf("create user %d: %v", index, err)
		}
	}
	for index, itemID := range items {
		if _, err := pool.Exec(ctx, `
			INSERT INTO items (id, owner_id, category_id, title, photo_urls)
			VALUES (
				$1,
				$2,
				(SELECT id FROM categories WHERE slug = 'books'),
				$3,
				ARRAY['https://example.com/notification.jpg']
			)`,
			itemID,
			users[index],
			"Notification item "+[]string{"A", "B"}[index],
		); err != nil {
			t.Fatalf("create item %d: %v", index, err)
		}
	}

	if _, err := pool.Exec(
		ctx,
		"INSERT INTO chains (id, signature, composition_key) VALUES ($1, $2, $3)",
		exchangeID,
		"notifications:"+exchangeID.String(),
		"notifications:"+exchangeID.String(),
	); err != nil {
		t.Fatalf("create exchange: %v", err)
	}
	for index := range items {
		if _, err := pool.Exec(ctx, `
			INSERT INTO chain_participants
				(chain_id, user_id, gives_item_id, receives_item_id, position)
			VALUES ($1, $2, $3, $4, $5)`,
			exchangeID,
			users[index],
			items[index],
			items[(index+1)%len(items)],
			index,
		); err != nil {
			t.Fatalf("create participant %d: %v", index, err)
		}
	}

	var messageID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO chain_messages (chain_id, author_id, body)
		VALUES ($1, $2, 'Привезу завтра')
		RETURNING id`,
		exchangeID,
		users[0],
	).Scan(&messageID); err != nil {
		t.Fatalf("create participant message: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO chain_messages (chain_id, kind)
		VALUES ($1, 'exchange_confirmed')`,
		exchangeID,
	); err != nil {
		t.Fatalf("create exchange event: %v", err)
	}

	router := chi.NewRouter()
	service := notificationservice.New(notificationrepository.New(db.New(pool)))
	New(service).RegisterRoutes(router, func(next http.Handler) http.Handler { return next })

	request := func(method string, path string, userID uuid.UUID) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r = r.WithContext(authcontext.WithUserID(r.Context(), userID))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}

	// Автор не получает уведомление о собственном тексте: только о предложении и общем событии.
	alicePage := listNotificationPage(t, request(http.MethodGet, "/notifications", users[0]))
	if alicePage.UnreadCount != 2 || len(alicePage.Notifications) != 2 {
		t.Fatalf("author page = %+v, want two notifications", alicePage)
	}

	// Получатель видит предложение, чужой текст и системное событие.
	bobPage := listNotificationPage(t, request(http.MethodGet, "/notifications?unread_only=true", users[1]))
	if bobPage.UnreadCount != 3 || len(bobPage.Notifications) != 3 {
		t.Fatalf("recipient page = %+v, want three notifications", bobPage)
	}
	if bobPage.Notifications[0].Kind != "exchange_confirmed" ||
		bobPage.Notifications[1].Kind != "text" ||
		bobPage.Notifications[1].MessageID == nil ||
		*bobPage.Notifications[1].MessageID != messageID.String() ||
		bobPage.Notifications[2].Kind != "exchange_proposed" {
		t.Fatalf("notification order and context = %+v", bobPage.Notifications)
	}

	textNotificationID := bobPage.Notifications[1].ID
	marked := request(
		http.MethodPost,
		"/notifications/"+textNotificationID+"/read",
		users[1],
	)
	if marked.Code != http.StatusNoContent {
		t.Fatalf("mark one: status = %d, body = %s", marked.Code, marked.Body.String())
	}
	if page := listNotificationPage(t, request(http.MethodGet, "/notifications", users[1])); page.UnreadCount != 2 {
		t.Fatalf("unread after mark one = %d, want 2", page.UnreadCount)
	}

	// Чужое уведомление нельзя ни увидеть через свой список, ни отметить прочитанным.
	foreign := request(
		http.MethodPost,
		"/notifications/"+textNotificationID+"/read",
		users[2],
	)
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign mark: status = %d, want %d", foreign.Code, http.StatusNotFound)
	}

	allMarked := request(http.MethodPost, "/notifications/read-all", users[1])
	if allMarked.Code != http.StatusOK {
		t.Fatalf("mark all: status = %d, body = %s", allMarked.Code, allMarked.Body.String())
	}
	if page := listNotificationPage(t, request(http.MethodGet, "/notifications?unread_only=true", users[1])); page.UnreadCount != 0 || len(page.Notifications) != 0 {
		t.Fatalf("unread page after mark all = %+v", page)
	}
}

func listNotificationPage(t *testing.T, response *httptest.ResponseRecorder) notificationdto.PageResponse {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("list notifications: status = %d, body = %s", response.Code, response.Body.String())
	}
	var page notificationdto.PageResponse
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatalf("decode notification page: %v", err)
	}
	return page
}
