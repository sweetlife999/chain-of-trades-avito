//go:build integration

package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	supportmodel "github.com/sweetlife999/chain-of-trades-avito/internal/support/model"
)

// TestAdminSupportQueueIntegration держит превью очереди модерации на живой БД.
//
// Тест написан по факту падения: превью брало последнее сообщение, автор которого не
// администратор, и на обращении, которое администратор открыл сам, LATERAL не давал ни
// одной строки. sqlc типизирует last_message_body как NOT NULL по исходному столбцу и
// нуллабельность от LEFT JOIN не моделирует, поэтому один такой тред отдавал 500 на всю
// страницу очереди — не на свою строку, а на весь список. Подставным репозиторием это не
// поймать: ломалось на скане из базы.
func TestAdminSupportQueueIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	// Автор обращения — сам администратор: ровно тот случай, на котором падало.
	adminID := uuid.New()

	t.Cleanup(func() {
		cleanup := context.Background()
		if _, err := pool.Exec(cleanup,
			`DELETE FROM support_messages WHERE thread_id IN
			 (SELECT id FROM support_threads WHERE user_id = $1)`, adminID); err != nil {
			t.Errorf("cleanup support messages: %v", err)
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE id = $1", adminID); err != nil {
			t.Errorf("cleanup users: %v", err)
		}
		pool.Close()
	})

	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, nickname, password_hash, is_admin)
		 VALUES ($1, $2, 'no-login', true)`,
		adminID, "queue-admin-"+adminID.String()[:8],
	)
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	repository := New(db.New(pool))

	thread, err := repository.Create(ctx, adminID, "Очередь модерации", "Первое сообщение автора")
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	// Автоответчик отвечает почти сразу — и до правки именно его ответ вытеснял вопрос
	// из превью очереди, а на этом треде оставлял превью пустым. Ник строкой, а не
	// константой из service: тот пакет импортирует этот, и ссылка назад дала бы цикл.
	if _, err := repository.CreateBotMessage(ctx, thread.ID, "У-бот", "автоответ"); err != nil {
		t.Fatalf("create bot message: %v", err)
	}

	page, _, err := repository.ListAdmin(ctx, supportmodel.AdminFilter{Limit: 100})
	if err != nil {
		t.Fatalf("list admin queue: %v", err)
	}

	var found *supportmodel.Thread
	for index := range page {
		if page[index].ID == thread.ID {
			found = &page[index]
			break
		}
	}
	if found == nil {
		t.Fatalf("thread %s is missing from the admin queue", thread.ID)
	}
	if found.LastMessage != "Первое сообщение автора" {
		t.Errorf(
			"queue preview = %q, want the thread author's own message",
			found.LastMessage,
		)
	}
	if found.EscalatedAt != nil {
		t.Errorf("thread is escalated at %v, want no mark before the bot fails", *found.EscalatedAt)
	}

	// Бот ответил, пользователь не возражал — модератору тут делать нечего, и в очереди
	// «ждут человека» такого обращения быть не должно.
	if _, listed := findThread(t, ctx, repository, thread.ID, true); listed {
		t.Error("answered thread is in the needs-human queue before any escalation")
	}

	// Эскалация: бот не справился и позвал человека. Очередь обязана показать метку —
	// по ней в админке рисуется пометка «нужен человек».
	if err := repository.Escalate(ctx, thread.ID); err != nil {
		t.Fatalf("escalate thread: %v", err)
	}
	escalated, listed := findThread(t, ctx, repository, thread.ID, false)
	if !listed {
		t.Fatalf("thread %s is missing from the admin queue", thread.ID)
	}
	if escalated.EscalatedAt == nil {
		t.Fatal("thread is not escalated after Escalate")
	}
	if _, listed := findThread(t, ctx, repository, thread.ID, true); !listed {
		t.Error("escalated thread is missing from the needs-human queue")
	}

	// Идемпотентность: повторный вызов не двигает метку, иначе «когда стало нужно
	// вмешаться» превратилось бы в «когда пользователь написал в последний раз».
	if err := repository.Escalate(ctx, thread.ID); err != nil {
		t.Fatalf("escalate thread twice: %v", err)
	}
	again, _ := findThread(t, ctx, repository, thread.ID, false)
	if !again.EscalatedAt.Equal(*escalated.EscalatedAt) {
		t.Errorf("escalation moved from %v to %v", *escalated.EscalatedAt, *again.EscalatedAt)
	}
}

// findThread ищет обращение в очереди модерации. Второе значение — нашлось ли оно: под
// фильтром needsHuman отсутствие обращения такой же ожидаемый исход, как присутствие.
func findThread(
	t *testing.T,
	ctx context.Context,
	repository *Repository,
	threadID uuid.UUID,
	needsHuman bool,
) (supportmodel.Thread, bool) {
	t.Helper()

	page, _, err := repository.ListAdmin(ctx, supportmodel.AdminFilter{
		Limit: 100, NeedsHuman: needsHuman,
	})
	if err != nil {
		t.Fatalf("list admin queue: %v", err)
	}
	for index := range page {
		if page[index].ID == threadID {
			return page[index], true
		}
	}
	return supportmodel.Thread{}, false
}
