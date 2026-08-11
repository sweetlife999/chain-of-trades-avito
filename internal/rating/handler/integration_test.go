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
	ratingdto "github.com/sweetlife999/chain-of-trades-avito/internal/rating/dto"
	ratingrepository "github.com/sweetlife999/chain-of-trades-avito/internal/rating/repository"
	ratingservice "github.com/sweetlife999/chain-of-trades-avito/internal/rating/service"
)

// TestRatingsIntegration проверяет оценку партнёра на живой БД: кого именно назначает
// партнёром сам цикл, перезапись оценки, пересчёт среднего балла триггером и три отказа —
// постороннему, на незавершённом обмене и по истечении срока. Правила живут в SQL и в
// схеме, поэтому подделать их фейком нельзя: только настоящая база.
func TestRatingsIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	// alice -> bob -> vera -> alice: alice отдаёт вещь A и получает B, значит оценивает
	// bob'а, чья вещь B к ней и пришла.
	users := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	items := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	completed := uuid.New()
	proposed := uuid.New()
	outsider := uuid.New()

	t.Cleanup(func() {
		cleanup := context.Background()
		if _, err := pool.Exec(cleanup, "DELETE FROM chains WHERE id = ANY($1)",
			[]uuid.UUID{completed, proposed}); err != nil {
			t.Errorf("cleanup chains: %v", err)
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM items WHERE id = ANY($1)", items); err != nil {
			t.Errorf("cleanup items: %v", err)
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE id = ANY($1)",
			append(users, outsider)); err != nil {
			t.Errorf("cleanup users: %v", err)
		}
		pool.Close()
	})

	for _, userID := range append(append([]uuid.UUID{}, users...), outsider) {
		if _, err := pool.Exec(
			ctx,
			"INSERT INTO users (id, nickname, password_hash) VALUES ($1, $2, $3)",
			userID, "rating-"+userID.String()[:8], "not-used-in-integration-test",
		); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	for index, itemID := range items {
		if _, err := pool.Exec(ctx, `
			INSERT INTO items (id, owner_id, category_id, title, photo_urls)
			VALUES ($1, $2, (SELECT id FROM categories WHERE slug = 'books'), $3,
			        ARRAY['https://example.com/rating.jpg'])`,
			itemID, users[index], "Rating item "+itemID.String()[:8],
		); err != nil {
			t.Fatalf("create item: %v", err)
		}
	}

	createChain := func(chainID uuid.UUID, status string) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO chains (id, signature, composition_key, status, closed_at)
			VALUES ($1, $2, $3, $4::chain_status,
			        CASE WHEN $4 = 'completed' THEN now() ELSE NULL END)`,
			chainID, "ratings:"+chainID.String(), "ratings:"+chainID.String(), status,
		); err != nil {
			t.Fatalf("create chain %s: %v", status, err)
		}
		for index := range items {
			if _, err := pool.Exec(ctx, `
				INSERT INTO chain_participants
					(chain_id, user_id, gives_item_id, receives_item_id, position)
				VALUES ($1, $2, $3, $4, $5)`,
				chainID, users[index], items[index], items[(index+1)%len(items)], index,
			); err != nil {
				t.Fatalf("create participant: %v", err)
			}
		}
	}
	createChain(completed, "completed")
	createChain(proposed, "proposed")

	router := chi.NewRouter()
	service := ratingservice.New(ratingrepository.New(db.New(pool)))
	New(service).RegisterRoutes(router, func(next http.Handler) http.Handler { return next })

	rate := func(chainID uuid.UUID, userID uuid.UUID, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPut, "/exchanges/"+chainID.String()+"/rating",
			strings.NewReader(body))
		r = r.WithContext(authcontext.WithUserID(r.Context(), userID))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	profile := func(userID uuid.UUID) (rating *float64, count int32) {
		if err := pool.QueryRow(ctx,
			"SELECT rating::double precision, ratings_count FROM users WHERE id = $1", userID,
		).Scan(&rating, &count); err != nil {
			t.Fatalf("read profile: %v", err)
		}
		return rating, count
	}

	// Партнёра назначает цикл: alice получила вещь bob'а, значит оценивает его.
	response := rate(completed, users[0], `{"score":4,"comment":"всё вовремя"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("rate: код %d, ожидали 200: %s", response.Code, response.Body.String())
	}
	var stored ratingdto.RatingResponse
	if err := json.NewDecoder(response.Body).Decode(&stored); err != nil {
		t.Fatalf("decode rating: %v", err)
	}
	if stored.RatedUserID != users[1].String() {
		t.Fatalf("оценён %s, а вещь пришла от %s", stored.RatedUserID, users[1])
	}

	if rating, count := profile(users[1]); rating == nil || *rating != 4 || count != 1 {
		t.Fatalf("после первой оценки профиль = %v/%d, ожидали 4/1", rating, count)
	}

	// Перезапись: строка одна, среднее пересчитано, второй оценки не появилось.
	if response := rate(completed, users[0], `{"score":2,"comment":"передумала"}`); response.Code != http.StatusOK {
		t.Fatalf("повторная оценка: код %d, ожидали 200", response.Code)
	}
	if rating, count := profile(users[1]); rating == nil || *rating != 2 || count != 1 {
		t.Fatalf("после правки профиль = %v/%d, ожидали 2/1", rating, count)
	}

	// Вторая оценка тому же человеку из другой цепочки невозможна, зато его оценивает
	// другой участник этой: bob получил вещь vera, vera — вещь alice.
	if response := rate(completed, users[2], `{"score":5,"comment":""}`); response.Code != http.StatusOK {
		t.Fatalf("оценка от третьего участника: код %d, ожидали 200", response.Code)
	}
	if rating, count := profile(users[0]); rating == nil || *rating != 5 || count != 1 {
		t.Fatalf("профиль alice = %v/%d, ожидали 5/1", rating, count)
	}

	// Лента анонимна и отдаёт только балл, текст и дату.
	feed := httptest.NewRecorder()
	router.ServeHTTP(feed, httptest.NewRequest(http.MethodGet, "/users/"+users[1].String()+"/ratings", nil))
	if feed.Code != http.StatusOK {
		t.Fatalf("лента: код %d, ожидали 200", feed.Code)
	}
	var page ratingdto.RatingsPageResponse
	if err := json.NewDecoder(feed.Body).Decode(&page); err != nil {
		t.Fatalf("decode feed: %v", err)
	}
	if len(page.Ratings) != 1 || page.Ratings[0].Score != 2 || page.Ratings[0].Comment != "передумала" {
		t.Fatalf("лента вернулась не той: %+v", page.Ratings)
	}

	// Посторонний в цепочке не участвует — 403, а не 404: обмен существует.
	if response := rate(completed, outsider, `{"score":5}`); response.Code != http.StatusForbidden {
		t.Fatalf("посторонний: код %d, ожидали 403", response.Code)
	}
	// Несуществующий обмен — 404.
	if response := rate(uuid.New(), users[0], `{"score":5}`); response.Code != http.StatusNotFound {
		t.Fatalf("несуществующий обмен: код %d, ожидали 404", response.Code)
	}
	// Незавершённый обмен оценивать нечего.
	if response := rate(proposed, users[0], `{"score":5}`); response.Code != http.StatusConflict {
		t.Fatalf("незавершённый обмен: код %d, ожидали 409", response.Code)
	}

	// Срок вышел — закрывается и правка уже поставленной оценки, а не только первая.
	if _, err := pool.Exec(ctx,
		"UPDATE chains SET closed_at = now() - interval '15 days' WHERE id = $1", completed,
	); err != nil {
		t.Fatalf("expire rating window: %v", err)
	}
	if response := rate(completed, users[0], `{"score":1,"comment":"месть"}`); response.Code != http.StatusConflict {
		t.Fatalf("просроченное окно: код %d, ожидали 409", response.Code)
	}
	if rating, count := profile(users[1]); rating == nil || *rating != 2 || count != 1 {
		t.Fatalf("после отказа профиль изменился: %v/%d, ожидали 2/1", rating, count)
	}
}
