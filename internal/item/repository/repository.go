package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
)

var (
	ErrNotFound                 = errors.New("item not found")
	ErrForbidden                = errors.New("item belongs to another user")
	ErrUnknownCategory          = errors.New("unknown category")
	ErrItemInChain              = errors.New("item participates in a chain")
	ErrSearchVisibilityConflict = errors.New("item status does not allow changing search visibility")
)

type Repository struct {
	pool         *pgxpool.Pool
	queries      *db.Queries
	transactions searchVisibilityTransactionManager
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:         pool,
		queries:      db.New(pool),
		transactions: &pgxSearchVisibilityTransactionManager{pool: pool},
	}
}

// Вещь и её желания пишутся двумя запросами, поэтому они идут одной транзакцией:
// иначе неизвестный слаг в wants оставил бы половину объявления в БД при ответе 400.
func (r *Repository) Create(ctx context.Context, item itemmodel.NewItem) (itemmodel.Item, error) {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return itemmodel.Item{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	queries := r.queries.WithTx(transaction)

	id, err := queries.InsertItem(ctx, db.InsertItemParams{
		OwnerID:     pgUUID(item.OwnerID),
		Category:    item.Category,
		Title:       item.Title,
		Description: item.Description,
		PhotoUrls:   item.PhotoURLs,
	})
	if err != nil {
		return itemmodel.Item{}, fmt.Errorf("insert item: %w", translateError(err))
	}

	if err := queries.InsertItemWants(ctx, db.InsertItemWantsParams{ItemID: id, Wants: item.Wants}); err != nil {
		return itemmodel.Item{}, fmt.Errorf("insert item wants: %w", translateError(err))
	}

	return r.commit(ctx, transaction, queries, id)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (itemmodel.Item, error) {
	found, err := r.queries.GetItemByID(ctx, pgUUID(id))
	if err != nil {
		return itemmodel.Item{}, fmt.Errorf("get item by id: %w", translateError(err))
	}

	return toModel(found), nil
}

// Строка списка и строка карточки — один набор колонок, поэтому конверсия типа вместо
// второго маппера. Разойдутся колонки — сломается сборка, а не ответ.
func (r *Repository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]itemmodel.Item, error) {
	rows, err := r.queries.ListItemsByOwner(ctx, pgUUID(ownerID))
	if err != nil {
		return nil, fmt.Errorf("list items by owner: %w", err)
	}

	items := make([]itemmodel.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, toModel(db.GetItemByIDRow(row)))
	}

	return items, nil
}

func (r *Repository) HasOpenExchange(ctx context.Context, id uuid.UUID) (bool, error) {
	hasOpenExchange, err := r.queries.ItemHasOpenExchange(ctx, pgUUID(id))
	if err != nil {
		return false, fmt.Errorf("check item open exchange: %w", err)
	}

	return hasOpenExchange, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, changes itemmodel.Changes) (itemmodel.Item, error) {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return itemmodel.Item{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	queries := r.queries.WithTx(transaction)

	if err := queries.UpdateItem(ctx, db.UpdateItemParams{
		Title:       optionalText(changes.Title),
		Description: optionalText(changes.Description),
		Category:    optionalText(changes.Category),
		PhotoUrls:   changes.PhotoURLs,
		ID:          pgUUID(id),
	}); err != nil {
		return itemmodel.Item{}, fmt.Errorf("update item: %w", translateError(err))
	}

	// Желания заменяются целиком: список коротких слагов, разбирать разницу дороже,
	// чем переписать. Оба запроса — в той же транзакции, промежуточного состояния «без
	// желаний» никто снаружи не увидит.
	if changes.Wants != nil {
		if err := queries.DeleteItemWants(ctx, pgUUID(id)); err != nil {
			return itemmodel.Item{}, fmt.Errorf("delete item wants: %w", translateError(err))
		}
		if err := queries.InsertItemWants(ctx, db.InsertItemWantsParams{
			ItemID: pgUUID(id),
			Wants:  changes.Wants,
		}); err != nil {
			return itemmodel.Item{}, fmt.Errorf("insert item wants: %w", translateError(err))
		}
	}

	return r.commit(ctx, transaction, queries, pgUUID(id))
}

// ClearPickupPoint возвращает вещь домой. Идемпотентен: у вещи без отметки и у вещи вне
// оборота снимать нечего, и повтор запроса — то же самое, что первый.
// Отметку ставит exchange-репозиторий: там она может утянуть за собой переход обмена в
// доставку, а снять её можно только когда обмена нет, — тянуть нечего.
func (r *Repository) ClearPickupPoint(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	if _, err := r.queries.ClearItemPickupPoint(ctx, db.ClearItemPickupPointParams{
		ItemID:  pgUUID(id),
		OwnerID: pgUUID(ownerID),
	}); err != nil {
		return fmt.Errorf("clear item pickup point: %w", err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := r.queries.DeleteItem(ctx, pgUUID(id))
	if err != nil {
		return fmt.Errorf("delete item: %w", translateError(err))
	}
	if deleted == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) ListCategories(ctx context.Context) ([]itemmodel.Category, error) {
	rows, err := r.queries.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	categories := make([]itemmodel.Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, itemmodel.Category{Slug: row.Slug, Name: row.Name})
	}

	return categories, nil
}

// Итоговое состояние читаем внутри транзакции: только так в ответ попадают и слаг
// категории, и желания, собранные из трёх таблиц.
func (r *Repository) commit(
	ctx context.Context,
	transaction pgx.Tx,
	queries *db.Queries,
	id pgtype.UUID,
) (itemmodel.Item, error) {
	saved, err := queries.GetItemByID(ctx, id)
	if err != nil {
		return itemmodel.Item{}, fmt.Errorf("read saved item: %w", translateError(err))
	}

	if err := transaction.Commit(ctx); err != nil {
		return itemmodel.Item{}, fmt.Errorf("commit transaction: %w", err)
	}

	return toModel(saved), nil
}

func optionalText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}

func toModel(item db.GetItemByIDRow) itemmodel.Item {
	return itemmodel.Item{
		ID:          uuid.UUID(item.ID.Bytes),
		OwnerID:     uuid.UUID(item.OwnerID.Bytes),
		Category:    item.Category,
		Title:       item.Title,
		Description: item.Description,
		PhotoURLs:   item.PhotoUrls,
		Wants:       item.Wants,
		Status:      string(item.Status),
		PickupPoint: toPickupPoint(item),
		CreatedAt:   item.CreatedAt.Time,
		UpdatedAt:   item.UpdatedAt.Time,
	}
}

// Пункт приезжает LEFT JOIN-ом, поэтому его отсутствие — это NULL в колонке, а не
// отдельный признак: невалидный id и означает «вещь дома».
func toPickupPoint(item db.GetItemByIDRow) *itemmodel.PickupPoint {
	if !item.PickupPointID.Valid {
		return nil
	}

	return &itemmodel.PickupPoint{
		ID:      uuid.UUID(item.PickupPointID.Bytes),
		Name:    item.PickupPointName.String,
		Address: item.PickupPointAddress.String,
	}
}

func translateError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		// Неизвестный слаг разыменовывается в NULL и упирается в NOT NULL — и у самой
		// вещи, и у её желаний колонка называется одинаково.
		case pgErr.Code == "23502" && pgErr.ColumnName == "category_id":
			return ErrUnknownCategory
		// Единственный внешний ключ, который может выстрелить на живой вещи, —
		// chain_participants с ON DELETE RESTRICT: вещь занята в цепочке обмена.
		case pgErr.Code == "23503":
			return ErrItemInChain
		}
	}

	return err
}
