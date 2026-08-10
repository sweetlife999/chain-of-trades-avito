package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
)

type searchVisibilityQueries interface {
	LockItemSearchVisibility(context.Context, pgtype.UUID) error
	GetItemSearchVisibilityForUpdate(
		context.Context,
		pgtype.UUID,
	) (db.GetItemSearchVisibilityForUpdateRow, error)
	WithdrawAvailableItem(context.Context, db.WithdrawAvailableItemParams) (int64, error)
	PublishWithdrawnItem(context.Context, db.PublishWithdrawnItemParams) (int64, error)
	CancelProposedExchangesForItemWithdrawal(
		context.Context,
		pgtype.UUID,
	) ([]db.CancelProposedExchangesForItemWithdrawalRow, error)
	GetItemByID(context.Context, pgtype.UUID) (db.GetItemByIDRow, error)
}

type searchVisibilityTransactionManager interface {
	WithinTransaction(context.Context, func(searchVisibilityQueries) error) error
}

type pgxSearchVisibilityTransactionManager struct {
	pool interface {
		Begin(context.Context) (pgx.Tx, error)
	}
}

func (m *pgxSearchVisibilityTransactionManager) WithinTransaction(
	ctx context.Context,
	operation func(searchVisibilityQueries) error,
) error {
	transaction, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin search visibility transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback(ctx)
		}
	}()

	if err := operation(db.New(transaction)); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit search visibility transaction: %w", err)
	}
	committed = true

	return nil
}

// SetSearchVisibility atomically changes whether an item participates in DFS. Disabling
// search also closes every proposed exchange containing the item. Confirmed exchanges are
// protected by the shared advisory lock and rejected by the status check.
func (r *Repository) SetSearchVisibility(
	ctx context.Context,
	itemID uuid.UUID,
	ownerID uuid.UUID,
	enabled bool,
) (itemmodel.SearchVisibilityChange, error) {
	var result itemmodel.SearchVisibilityChange
	err := r.transactions.WithinTransaction(ctx, func(queries searchVisibilityQueries) error {
		change, err := setSearchVisibility(ctx, queries, itemID, ownerID, enabled)
		if err != nil {
			return err
		}
		result = change
		return nil
	})
	if err != nil {
		return itemmodel.SearchVisibilityChange{}, fmt.Errorf("set item search visibility: %w", err)
	}

	return result, nil
}

func setSearchVisibility(
	ctx context.Context,
	queries searchVisibilityQueries,
	itemID uuid.UUID,
	ownerID uuid.UUID,
	enabled bool,
) (itemmodel.SearchVisibilityChange, error) {
	databaseItemID := pgUUID(itemID)
	databaseOwnerID := pgUUID(ownerID)
	if err := queries.LockItemSearchVisibility(ctx, databaseItemID); err != nil {
		return itemmodel.SearchVisibilityChange{}, fmt.Errorf("lock item search visibility: %w", err)
	}

	current, err := queries.GetItemSearchVisibilityForUpdate(ctx, databaseItemID)
	if err != nil {
		return itemmodel.SearchVisibilityChange{}, fmt.Errorf("get item search visibility: %w", translateError(err))
	}
	if uuid.UUID(current.OwnerID.Bytes) != ownerID {
		return itemmodel.SearchVisibilityChange{}, ErrForbidden
	}

	result := itemmodel.SearchVisibilityChange{}
	if enabled {
		if current.Status == db.ItemStatusAvailable {
			return readVisibilityResult(ctx, queries, databaseItemID, result)
		}
		if current.Status != db.ItemStatusWithdrawn {
			return itemmodel.SearchVisibilityChange{}, ErrSearchVisibilityConflict
		}

		updated, err := queries.PublishWithdrawnItem(ctx, db.PublishWithdrawnItemParams{
			ItemID:  databaseItemID,
			OwnerID: databaseOwnerID,
		})
		if err != nil {
			return itemmodel.SearchVisibilityChange{}, fmt.Errorf("publish item to search: %w", err)
		}
		if updated != 1 {
			return itemmodel.SearchVisibilityChange{}, ErrSearchVisibilityConflict
		}
		result.Changed = true
		return readVisibilityResult(ctx, queries, databaseItemID, result)
	}

	if current.Status == db.ItemStatusWithdrawn {
		return readVisibilityResult(ctx, queries, databaseItemID, result)
	}
	if current.Status != db.ItemStatusAvailable {
		return itemmodel.SearchVisibilityChange{}, ErrSearchVisibilityConflict
	}

	updated, err := queries.WithdrawAvailableItem(ctx, db.WithdrawAvailableItemParams{
		ItemID:  databaseItemID,
		OwnerID: databaseOwnerID,
	})
	if err != nil {
		return itemmodel.SearchVisibilityChange{}, fmt.Errorf("withdraw item from search: %w", err)
	}
	if updated != 1 {
		return itemmodel.SearchVisibilityChange{}, ErrSearchVisibilityConflict
	}

	recoveryRows, err := queries.CancelProposedExchangesForItemWithdrawal(ctx, databaseItemID)
	if err != nil {
		return itemmodel.SearchVisibilityChange{}, fmt.Errorf("cancel item proposals: %w", err)
	}
	result.Changed = true
	result.RecoveryCandidates = make([]itemmodel.SearchCandidate, len(recoveryRows))
	for index, row := range recoveryRows {
		result.RecoveryCandidates[index] = itemmodel.SearchCandidate{
			ItemID:  uuid.UUID(row.ItemID.Bytes),
			OwnerID: uuid.UUID(row.OwnerID.Bytes),
		}
	}

	return readVisibilityResult(ctx, queries, databaseItemID, result)
}

func readVisibilityResult(
	ctx context.Context,
	queries searchVisibilityQueries,
	itemID pgtype.UUID,
	result itemmodel.SearchVisibilityChange,
) (itemmodel.SearchVisibilityChange, error) {
	item, err := queries.GetItemByID(ctx, itemID)
	if err != nil {
		return itemmodel.SearchVisibilityChange{}, fmt.Errorf("read item after search visibility change: %w", translateError(err))
	}
	result.Item = toModel(item)

	return result, nil
}
