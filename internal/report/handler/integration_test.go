//go:build integration

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	authcontext "github.com/sweetlife999/chain-of-trades-avito/internal/auth/authcontext"
	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	reportdto "github.com/sweetlife999/chain-of-trades-avito/internal/report/dto"
	reportrepository "github.com/sweetlife999/chain-of-trades-avito/internal/report/repository"
	reportservice "github.com/sweetlife999/chain-of-trades-avito/internal/report/service"
)

// TestReportsIntegration гоняет POST /reports на живой БД: успешную жалобу и все четыре
// запрета. Два из них держит база (UNIQUE на повтор, отсутствие строки на несуществующее
// сообщение), и подставным репозиторием их не проверить.
func TestReportsIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	// Четвёртый пользователь в цепочку не входит: он проверяет запрет для постороннего.
	users := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	items := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	chainID := uuid.New()

	t.Cleanup(func() {
		cleanup := context.Background()
		if _, err := pool.Exec(cleanup, "DELETE FROM chains WHERE id = $1", chainID); err != nil {
			t.Errorf("cleanup chains: %v", err)
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
		_, err := pool.Exec(
			ctx,
			"INSERT INTO users (id, nickname, password_hash) VALUES ($1, $2, $3)",
			userID,
			"reports-"+userID.String()[:8],
			"not-used-in-integration-test",
		)
		if err != nil {
			t.Fatalf("create user %d: %v", index, err)
		}
	}

	for index, itemID := range items {
		_, err := pool.Exec(ctx, `
			INSERT INTO items (id, owner_id, category_id, title, photo_urls)
			VALUES (
				$1,
				$2,
				(SELECT id FROM categories WHERE slug = 'books'),
				$3,
				ARRAY['https://example.com/reports.jpg']
			)`,
			itemID,
			users[index],
			"Reports integration item",
		)
		if err != nil {
			t.Fatalf("create item %d: %v", index, err)
		}
	}

	if _, err := pool.Exec(
		ctx,
		"INSERT INTO chains (id, signature) VALUES ($1, $2)",
		chainID,
		"reports:"+chainID.String(),
	); err != nil {
		t.Fatalf("create chain: %v", err)
	}

	for index := range items {
		_, err := pool.Exec(ctx, `
			INSERT INTO chain_participants
				(chain_id, user_id, gives_item_id, receives_item_id, position)
			VALUES ($1, $2, $3, $4, $5)`,
			chainID,
			users[index],
			items[index],
			items[(index+1)%len(items)],
			index,
		)
		if err != nil {
			t.Fatalf("create participant %d: %v", index, err)
		}
	}

	var messageID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO chain_messages (chain_id, author_id, body)
		VALUES ($1, $2, $3)
		RETURNING id`,
		chainID, users[1], "напишите мне в другом мессенджере, там договоримся",
	).Scan(&messageID); err != nil {
		t.Fatalf("create message: %v", err)
	}

	var eventID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO chain_messages (chain_id, kind)
		VALUES ($1, 'exchange_confirmed')
		RETURNING id`,
		chainID,
	).Scan(&eventID); err != nil {
		t.Fatalf("create event: %v", err)
	}

	reportsRepository := reportrepository.New(db.New(pool))
	router := chi.NewRouter()
	service := reportservice.New(reportsRepository)
	New(service).RegisterRoutes(router, func(next http.Handler) http.Handler { return next })

	post := func(reporter uuid.UUID, target uuid.UUID, reason string) *httptest.ResponseRecorder {
		body := `{"message_id":"` + target.String() + `","reason":"` + reason + `","comment":"уводит сделку"}`
		r := httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(body))
		r = r.WithContext(authcontext.WithUserID(r.Context(), reporter))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		return w
	}

	accepted := post(users[0], messageID, "spam")
	if accepted.Code != http.StatusCreated {
		t.Fatalf("жалоба участника: status = %d, want %d (body %s)",
			accepted.Code, http.StatusCreated, accepted.Body.String())
	}

	var created struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(accepted.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Status != "open" {
		t.Fatalf("status = %q, want open", created.Status)
	}
	if created.Comment != "уводит сделку" {
		t.Fatalf("comment = %q, want сохранённый комментарий", created.Comment)
	}

	cases := []struct {
		name     string
		reporter uuid.UUID
		target   uuid.UUID
		want     int
	}{
		{name: "повтор", reporter: users[0], target: messageID, want: http.StatusConflict},
		{name: "собственное сообщение", reporter: users[1], target: messageID, want: http.StatusForbidden},
		{name: "посторонний", reporter: users[3], target: messageID, want: http.StatusForbidden},
		{name: "событие обмена", reporter: users[0], target: eventID, want: http.StatusBadRequest},
		{name: "несуществующее сообщение", reporter: users[0], target: uuid.New(), want: http.StatusNotFound},
	}

	for _, test := range cases {
		w := post(test.reporter, test.target, "abuse")
		if w.Code != test.want {
			t.Errorf("%s: status = %d, want %d (body %s)",
				test.name, w.Code, test.want, w.Body.String())
		}
	}

	// Второй участник жалуется на то же сообщение: UNIQUE стоит на паре, а не на сообщении.
	if w := post(users[2], messageID, "abuse"); w.Code != http.StatusCreated {
		t.Fatalf("жалоба второго участника: status = %d, want %d (body %s)",
			w.Code, http.StatusCreated, w.Body.String())
	}

	// Админские чтения используют ту же живую базу: фильтр очереди, карточку и полный
	// тред. Middleware роли покрыт unit-тестом, здесь проверяем SQL-связи и JSON.
	adminRouter := chi.NewRouter()
	adminService := reportservice.NewAdmin(reportsRepository, exchangerepository.New(pool))
	NewAdmin(adminService).RegisterRoutes(adminRouter)

	queue := httptest.NewRecorder()
	adminRouter.ServeHTTP(
		queue,
		httptest.NewRequest(http.MethodGet, "/reports?status=open&reason=spam&limit=10", nil),
	)
	if queue.Code != http.StatusOK {
		t.Fatalf("admin queue: status = %d, want 200 (body %s)", queue.Code, queue.Body.String())
	}
	var page reportdto.AdminReportListResponse
	if err := json.NewDecoder(queue.Body).Decode(&page); err != nil {
		t.Fatalf("decode admin queue: %v", err)
	}
	if page.Pagination.Total != 1 || len(page.Reports) != 1 || page.Reports[0].ID != created.ID {
		t.Fatalf("admin queue = %+v", page)
	}

	detail := httptest.NewRecorder()
	adminRouter.ServeHTTP(
		detail,
		httptest.NewRequest(http.MethodGet, "/reports/"+created.ID, nil),
	)
	if detail.Code != http.StatusOK {
		t.Fatalf("admin detail: status = %d, want 200 (body %s)", detail.Code, detail.Body.String())
	}
	var reportDetail reportdto.AdminReportResponse
	if err := json.NewDecoder(detail.Body).Decode(&reportDetail); err != nil {
		t.Fatalf("decode admin detail: %v", err)
	}
	if reportDetail.Reporter.ID != users[0].String() ||
		reportDetail.Offender.ID != users[1].String() ||
		reportDetail.Exchange.ID != chainID.String() ||
		reportDetail.Message.ID != messageID.String() {
		t.Fatalf("admin detail = %+v", reportDetail)
	}

	thread := httptest.NewRecorder()
	adminRouter.ServeHTTP(
		thread,
		httptest.NewRequest(http.MethodGet, "/reports/"+created.ID+"/messages", nil),
	)
	if thread.Code != http.StatusOK {
		t.Fatalf("admin thread: status = %d, want 200 (body %s)", thread.Code, thread.Body.String())
	}
	var messages reportdto.AdminReportMessagesResponse
	if err := json.NewDecoder(thread.Body).Decode(&messages); err != nil {
		t.Fatalf("decode admin thread: %v", err)
	}
	if messages.ExchangeID != chainID.String() || len(messages.Messages) != 2 {
		t.Fatalf("admin thread = %+v", messages)
	}
}
