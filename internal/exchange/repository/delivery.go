package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

var ErrUnknownPickupPoint = errors.New("pickup point does not exist")

// RecordItemPickup запоминает, в какой пункт владелец отнёс вещь, и — если этим она была
// последней в подтверждённом обмене — переводит обмен в доставку. Сдать вещь можно и вне
// обмена: отметка живёт на вещи, а не на цепочке.
//
// Порядок блокировок здесь тот же, что в decision.go, completion.go и admin.go: сначала
// advisory на вещь, затем строка обмена, затем строки вещей. Первая не даёт подтверждению
// обмена проскочить между «есть ли у вещи обмен» и записью пункта, вторая выстраивает в
// очередь сдачи вещей соседями по цепочке — без неё двое одновременно увидели бы вещь
// друг друга ещё не сданной, и обмен остался бы в confirmed при всех вещах в пунктах.
func (r *Repository) RecordItemPickup(
	ctx context.Context,
	itemID uuid.UUID,
	ownerID uuid.UUID,
	pickupPointID uuid.UUID,
) error {
	err := r.transactions.WithinTransaction(ctx, func(queries exchangeWriteQueries) error {
		if err := queries.LockItemPickup(ctx, pgUUID(itemID)); err != nil {
			return fmt.Errorf("lock item pickup: %w", err)
		}

		// Вещь без подтверждённого обмена — обычное дело: её несут в пункт заранее.
		inExchange := true
		exchangeID, err := queries.FindConfirmedExchangeForItem(ctx, pgUUID(itemID))
		if errors.Is(err, pgx.ErrNoRows) {
			inExchange = false
		} else if err != nil {
			return fmt.Errorf("find confirmed exchange for item: %w", err)
		}

		if inExchange {
			exchange, err := queries.LockExchange(ctx, exchangeID)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			if err != nil {
				return fmt.Errorf("lock exchange for item pickup: %w", err)
			}
			// Пока ждали блокировку, обмен мог уехать: сосед сдал последнюю вещь и увёл
			// цепочку в доставку. Пункт вещи запоминаем всё равно, а событий больше нет.
			inExchange = exchange.Status == db.ChainStatusConfirmed
		}

		updated, err := queries.SetItemPickupPoint(ctx, db.SetItemPickupPointParams{
			PickupPointID: pgUUID(pickupPointID),
			ItemID:        pgUUID(itemID),
			OwnerID:       pgUUID(ownerID),
		})
		if err != nil {
			return translatePickupError(err)
		}
		// Владельца и существование вещи сервис проверяет заранее, поэтому здесь ноль строк
		// означает единственное оставшееся: вещь уже обменяна или снята с публикации.
		if updated == 0 {
			return ErrConflict
		}

		if !inExchange {
			return nil
		}

		err = recordExchangeEvent(
			ctx,
			queries,
			uuid.UUID(exchangeID.Bytes),
			pgUUID(ownerID),
			db.ChainMessageKindParticipantDeliveredItem,
		)
		if err != nil {
			return err
		}

		return promoteToDelivering(ctx, queries, uuid.UUID(exchangeID.Bytes))
	})
	if err != nil {
		return fmt.Errorf("record item pickup: %w", err)
	}

	return nil
}

// promoteToDelivering переводит обмен в доставку, если в пунктах уже лежат все его вещи.
// Вызывается из двух мест: сдачи вещи и последнего подтверждения участия. Второе нужно
// потому, что вещи можно отнести в пункт заранее — тогда к моменту сбора обмена сдавать
// уже нечего, и без этой проверки цепочка осталась бы в confirmed навсегда.
func promoteToDelivering(
	ctx context.Context,
	queries exchangeWriteQueries,
	exchangeID uuid.UUID,
) error {
	promoted, err := queries.PromoteExchangeToDelivering(ctx, pgUUID(exchangeID))
	if err != nil {
		return fmt.Errorf("promote exchange to delivering: %w", err)
	}
	if promoted == 0 {
		return nil
	}

	return recordExchangeEvent(
		ctx,
		queries,
		exchangeID,
		pgtype.UUID{},
		db.ChainMessageKindExchangeDelivering,
	)
}

// Единственный внешний ключ, который может выстрелить на этом UPDATE, — ссылка на
// pickup_points: пункт удалили или клиент прислал чужой UUID.
func translatePickupError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrUnknownPickupPoint
	}

	return fmt.Errorf("set item pickup point: %w", err)
}
