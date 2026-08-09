package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

// CancelByAdmin принудительно закрывает proposed или confirmed обмен одной
// транзакцией. Для confirmed объявления освобождаются; пользовательские решения и
// счётчики не меняются, потому что отмену инициировал администратор.
func (r *Repository) CancelByAdmin(
	ctx context.Context,
	exchangeID uuid.UUID,
) ([]exchangemodel.Node, string, error) {
	var (
		recoveryNodes []exchangemodel.Node
		signature     string
	)

	err := r.transactions.WithinTransaction(ctx, func(queries exchangeWriteQueries) error {
		// Тот же advisory lock используется при confirm/decline: он не позволяет
		// административной отмене одновременно резервировать те же объявления.
		if err := queries.LockExchangeDecisionItems(ctx, pgUUID(exchangeID)); err != nil {
			return fmt.Errorf("lock admin cancellation items: %w", err)
		}

		exchange, err := queries.LockExchange(ctx, pgUUID(exchangeID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("lock exchange for admin cancellation: %w", err)
		}
		// Отказаться участник может только до доставки, поэтому дальше обмен закрывает
		// один администратор — принимаются все четыре незакрытых статуса.
		switch exchange.Status {
		case db.ChainStatusProposed, db.ChainStatusConfirmed,
			db.ChainStatusDelivering, db.ChainStatusDelivered:
		default:
			return ErrConflict
		}
		signature = exchange.Signature

		items, err := queries.LockExchangeItems(ctx, pgUUID(exchangeID))
		if err != nil {
			return fmt.Errorf("lock exchange items for admin cancellation: %w", err)
		}
		if len(items) == 0 {
			return ErrConflict
		}

		// Вещи резервируются при подтверждении и остаются reserved до конца: ни доставка,
		// ни выдача их статус не меняют. Пункт хранения при отмене не сбрасываем — вещь
		// физически осталась там же, и владелец забирает её отдельным действием.
		expectedStatus := db.ItemStatusReserved
		if exchange.Status == db.ChainStatusProposed {
			expectedStatus = db.ItemStatusAvailable
		}
		for _, item := range items {
			if item.Status != expectedStatus {
				return ErrConflict
			}
			recoveryNodes = append(recoveryNodes, exchangemodel.Node{
				ItemID:  uuid.UUID(item.ID.Bytes),
				OwnerID: uuid.UUID(item.OwnerID.Bytes),
			})
		}

		if exchange.Status != db.ChainStatusProposed {
			released, err := queries.ReleaseExchangeItems(ctx, pgUUID(exchangeID))
			if err != nil {
				return fmt.Errorf("release admin-cancelled exchange items: %w", err)
			}
			if released != int64(len(items)) {
				return ErrConflict
			}
		}

		if err := queries.CancelExchange(ctx, pgUUID(exchangeID)); err != nil {
			return fmt.Errorf("cancel exchange by admin: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("admin cancel exchange: %w", err)
	}

	return recoveryNodes, signature, nil
}
