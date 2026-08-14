//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

// TestAntiscamDecideIntegration держит решение администратора по AI-подозрению на живой БД.
//
// Тест написан по факту падения: в UPDATE параметр решения сначала встречался как
// sqlc.arg(decision)::text, поэтому Postgres выводил его тип как text, а следом
// присваивание decision = $1 в колонку-enum не проходило разбор запроса. Ломался весь
// запрос, а не отдельная ветка: и «подтвердить нарушение», и «ложное срабатывание»
// отдавали 500. Подставным репозиторием это не поймать — падало в самой базе.
func TestAntiscamDecideIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	suspectID, adminID, chainID := uuid.New(), uuid.New(), uuid.New()

	t.Cleanup(func() {
		cleanup := context.Background()
		// Кейсы, ссылки на сообщения и сами сообщения уезжают каскадом за цепочкой.
		if _, err := pool.Exec(cleanup, "DELETE FROM chains WHERE id = $1", chainID); err != nil {
			t.Errorf("cleanup chain: %v", err)
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE id = ANY($1)",
			[]uuid.UUID{suspectID, adminID}); err != nil {
			t.Errorf("cleanup users: %v", err)
		}
		pool.Close()
	})

	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, nickname, password_hash, is_admin) VALUES
		 ($1, $2, 'no-login', false),
		 ($3, $4, 'no-login', true)`,
		suspectID, "antiscam-suspect-"+suspectID.String()[:8],
		adminID, "antiscam-admin-"+adminID.String()[:8],
	)
	if err != nil {
		t.Fatalf("seed users: %v", err)
	}
	// signature и composition_key уникальны только среди живых обменов, но NOT NULL —
	// всегда, поэтому id цепочки годится и туда.
	if _, err := pool.Exec(ctx,
		"INSERT INTO chains (id, signature, composition_key) VALUES ($1, $2, $2)",
		chainID, chainID.String()); err != nil {
		t.Fatalf("seed chain: %v", err)
	}

	var caseID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO antiscam_cases (chain_id, suspect_user_id, risk, category, reason)
		 VALUES ($1, $2, 80, 'external_payment', 'просит перевести на карту')
		 RETURNING id`,
		chainID, suspectID,
	).Scan(&caseID)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}

	// Улика — сообщение-событие без автора: карточке нужна хотя бы одна строка в
	// antiscam_case_messages, иначе LATERAL в Get не даёт ни одной. Текстовая реплика
	// потребовала бы участника цепочки с двумя вещами, а запросу здесь всё равно.
	var messageID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO chain_messages (chain_id, kind) VALUES ($1, 'exchange_confirmed') RETURNING id`,
		chainID,
	).Scan(&messageID)
	if err != nil {
		t.Fatalf("seed message: %v", err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO antiscam_case_messages (case_id, message_id) VALUES ($1, $2)",
		caseID, messageID); err != nil {
		t.Fatalf("seed evidence: %v", err)
	}

	repository := New(db.New(pool))

	if err := repository.Decide(ctx, caseID, adminID, "confirmed", "нарушение подтверждено"); err != nil {
		t.Fatalf("decide case: %v", err)
	}

	var (
		status, decision, comment string
		reviewedBy                uuid.UUID
		closedAt                  *time.Time
	)
	err = pool.QueryRow(ctx,
		`SELECT status::text, decision::text, resolution_comment, reviewed_by, closed_at
		 FROM antiscam_cases WHERE id = $1`, caseID,
	).Scan(&status, &decision, &comment, &reviewedBy, &closedAt)
	if err != nil {
		t.Fatalf("read case: %v", err)
	}
	if status != "resolved" {
		t.Errorf("status = %q, want resolved after a confirmed decision", status)
	}
	if decision != "confirmed" {
		t.Errorf("decision = %q, want confirmed", decision)
	}
	if comment != "нарушение подтверждено" {
		t.Errorf("resolution comment = %q, want the admin's comment", comment)
	}
	if reviewedBy != adminID {
		t.Errorf("reviewed by %s, want %s", reviewedBy, adminID)
	}
	if closedAt == nil {
		t.Error("closed_at is empty on a closed case")
	}

	// Второй администратор нажал ту же кнопку: карточка уже закрыта, и повтор обязан
	// отличаться от «не найдено» — на этом стоит 409 в handler.
	err = repository.Decide(ctx, caseID, adminID, "false_positive", "повторное решение")
	if !errors.Is(err, ErrAlreadyReviewed) {
		t.Errorf("second decide error = %v, want ErrAlreadyReviewed", err)
	}
}
