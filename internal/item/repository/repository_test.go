package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
)

// Проверяет то, что нельзя проверить фейком: коды ошибок PostgreSQL, из которых
// хэндлер выводит 400/409. Без живой БД тест пропускается — `make up`, и он оживает.
func TestRepositoryAgainstDatabase(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("database is not reachable: %v", err)
	}

	repository := New(pool)

	// Пользователь уносит за собой свои вещи (items.owner_id ON DELETE CASCADE),
	// поэтому убирать за тестом достаточно его одного.
	var ownerID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO users (nickname, password_hash) VALUES ($1, 'hash') RETURNING id`,
		"item-repo-test-"+uuid.NewString()[:8],
	).Scan(&ownerID)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID)
	})

	created, err := repository.Create(ctx, itemmodel.NewItem{
		OwnerID:     ownerID,
		Category:    "bikes",
		Title:       "Велосипед",
		Description: "Почти новый",
		PhotoURLs:   []string{"https://example.com/1.jpg", "https://example.com/2.jpg"},
		Wants:       []string{"consoles", "phones"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(created.PhotoURLs) != 2 || created.Category != "bikes" {
		t.Fatalf("Create() = %+v", created)
	}
	if len(created.Wants) != 2 || created.Wants[0] != "consoles" || created.Wants[1] != "phones" {
		t.Fatalf("Create() wants = %v", created.Wants)
	}
	if created.Status != "available" {
		t.Fatalf("Create() status = %q, want available", created.Status)
	}

	_, err = repository.Create(ctx, itemmodel.NewItem{
		OwnerID:   ownerID,
		Category:  "no-such-category",
		Title:     "Мимо справочника",
		PhotoURLs: []string{"https://example.com/1.jpg"},
		Wants:     []string{"books"},
	})
	if !errors.Is(err, ErrUnknownCategory) {
		t.Fatalf("Create() with unknown category error = %v, want ErrUnknownCategory", err)
	}

	if _, err := repository.Create(ctx, itemmodel.NewItem{
		OwnerID:   ownerID,
		Category:  "bikes",
		Title:     "Мимо справочника в желаниях",
		PhotoURLs: []string{"https://example.com/1.jpg"},
		Wants:     []string{"books", "no-such-category"},
	}); !errors.Is(err, ErrUnknownCategory) {
		t.Fatalf("Create() with unknown want error = %v, want ErrUnknownCategory", err)
	}

	newTitle := "Велосипед городской"
	updated, err := repository.Update(ctx, created.ID, itemmodel.Changes{
		Title:     &newTitle,
		PhotoURLs: []string{"https://example.com/3.jpg"},
		Wants:     []string{"tools"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != newTitle || len(updated.PhotoURLs) != 1 {
		t.Fatalf("Update() = %+v", updated)
	}
	if len(updated.Wants) != 1 || updated.Wants[0] != "tools" {
		t.Fatalf("Update() wants = %v, want [tools]", updated.Wants)
	}
	if updated.Category != "bikes" || updated.Description != "Почти новый" {
		t.Fatalf("Update() затронул поля, которые не передавали: %+v", updated)
	}

	// Провалившееся обновление не должно оставить половину изменений.
	brokenTitle := "Не должно сохраниться"
	if _, err := repository.Update(ctx, created.ID, itemmodel.Changes{
		Title: &brokenTitle,
		Wants: []string{"no-such-category"},
	}); !errors.Is(err, ErrUnknownCategory) {
		t.Fatalf("Update() with unknown want error = %v, want ErrUnknownCategory", err)
	}

	current, err := repository.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if current.Title != newTitle {
		t.Fatalf("транзакция не откатилась: title = %q, want %q", current.Title, newTitle)
	}
	if len(current.Wants) != 1 || current.Wants[0] != "tools" {
		t.Fatalf("транзакция не откатилась: wants = %v, want [tools]", current.Wants)
	}

	// Список владельца: чужие вещи в него не попадают, желания собираются так же,
	// как в карточке.
	var strangerID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO users (nickname, password_hash) VALUES ($1, 'hash') RETURNING id`,
		"item-repo-stranger-"+uuid.NewString()[:8],
	).Scan(&strangerID)
	if err != nil {
		t.Fatalf("create stranger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, strangerID)
	})
	if _, err := repository.Create(ctx, itemmodel.NewItem{
		OwnerID:   strangerID,
		Category:  "phones",
		Title:     "Чужой смартфон",
		PhotoURLs: []string{"https://example.com/stranger.jpg"},
		Wants:     []string{"bikes"},
	}); err != nil {
		t.Fatalf("Create() stranger item error = %v", err)
	}

	owned, err := repository.ListByOwner(ctx, ownerID)
	if err != nil {
		t.Fatalf("ListByOwner() error = %v", err)
	}
	if len(owned) != 1 || owned[0].ID != created.ID {
		t.Fatalf("ListByOwner() = %+v, want одну вещь %v", owned, created.ID)
	}
	if len(owned[0].Wants) != 1 || owned[0].Wants[0] != "tools" {
		t.Fatalf("ListByOwner() wants = %v, want [tools]", owned[0].Wants)
	}

	// Вещь, занятую в цепочке, удалить нельзя — из этой ошибки хэндлер делает 409.
	partner, err := repository.Create(ctx, itemmodel.NewItem{
		OwnerID:   ownerID,
		Category:  "phones",
		Title:     "Смартфон",
		PhotoURLs: []string{"https://example.com/phone.jpg"},
		Wants:     []string{"bikes"},
	})
	if err != nil {
		t.Fatalf("Create() partner item error = %v", err)
	}
	var chainID uuid.UUID
	if err := pool.QueryRow(
		ctx,
		`INSERT INTO chains (signature) VALUES ($1) RETURNING id`,
		"item-delete-test:"+uuid.NewString(),
	).Scan(&chainID); err != nil {
		t.Fatalf("create chain: %v", err)
	}
	// Живая цепочка держит и вещи, и пользователя (chain_participants.user_id RESTRICT),
	// поэтому её нужно убрать раньше, чем сработает уборка пользователя.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM chains WHERE id = $1`, chainID)
	})
	if _, err := pool.Exec(ctx,
		`INSERT INTO chain_participants (chain_id, user_id, gives_item_id, receives_item_id, position)
		 VALUES ($1, $2, $3, $4, 0)`,
		chainID, ownerID, created.ID, partner.ID,
	); err != nil {
		t.Fatalf("create chain participant: %v", err)
	}

	hasOpenExchange, err := repository.HasOpenExchange(ctx, created.ID)
	if err != nil {
		t.Fatalf("HasOpenExchange() error = %v", err)
	}
	if !hasOpenExchange {
		t.Fatal("HasOpenExchange() = false, want true")
	}

	if err := repository.Delete(ctx, created.ID); !errors.Is(err, ErrItemInChain) {
		t.Fatalf("Delete() of chained item error = %v, want ErrItemInChain", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM chains WHERE id = $1`, chainID); err != nil {
		t.Fatalf("delete chain: %v", err)
	}

	hasOpenExchange, err = repository.HasOpenExchange(ctx, created.ID)
	if err != nil {
		t.Fatalf("HasOpenExchange() after close error = %v", err)
	}
	if hasOpenExchange {
		t.Fatal("HasOpenExchange() after delete = true, want false")
	}

	if _, err := repository.GetByID(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() unknown id error = %v, want ErrNotFound", err)
	}
	if err := repository.Delete(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() unknown id error = %v, want ErrNotFound", err)
	}
	if err := repository.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}
