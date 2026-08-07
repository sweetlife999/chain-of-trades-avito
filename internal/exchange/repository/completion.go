package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

// CompleteParticipation подтверждает, что участник получил вещь.
// Последнее подтверждение атомарно завершает весь обмен.
func (r *Repository) CompleteParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	err := r.transactions.WithinTransaction(ctx, func(queries exchangeWriteQueries) error {
		// Тот же advisory lock используется при подтверждении участия. Благодаря этому
		// резервирование и завершение обмена не меняют одни вещи параллельно.
		if err := queries.LockExchangeDecisionItems(ctx, pgUUID(exchangeID)); err != nil {
			return fmt.Errorf("lock exchange completion items: %w", err)
		}

		exchangeStatus, err := queries.LockExchange(ctx, pgUUID(exchangeID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("lock exchange for completion: %w", err)
		}

		participant, err := queries.LockExchangeCompletionParticipant(
			ctx,
			db.LockExchangeCompletionParticipantParams{
				ExchangeID: pgUUID(exchangeID),
				UserID:     pgUUID(userID),
			},
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotParticipant
		}
		if err != nil {
			return fmt.Errorf("lock exchange completion participant: %w", err)
		}

		// Повторный запрос уже подтвердившего участника безопасен. Это важно,
		// например, если клиент не получил первый HTTP-ответ и отправил запрос снова.
		if participant.CompletionConfirmedAt.Valid &&
			(exchangeStatus == db.ChainStatusConfirmed || exchangeStatus == db.ChainStatusCompleted) {
			return nil
		}
		if exchangeStatus != db.ChainStatusConfirmed || participant.Status != db.ParticipantStatusAccepted {
			return ErrConflict
		}

		if err := queries.ConfirmExchangeParticipantCompletion(
			ctx,
			db.ConfirmExchangeParticipantCompletionParams{
				ExchangeID: pgUUID(exchangeID),
				UserID:     pgUUID(userID),
			},
		); err != nil {
			return fmt.Errorf("confirm exchange participant completion: %w", err)
		}

		err = recordExchangeEvent(
			ctx,
			queries,
			exchangeID,
			pgUUID(userID),
			db.ChainMessageKindParticipantCompleted,
		)
		if err != nil {
			return err
		}

		incomplete, err := queries.CountIncompleteExchangeParticipants(ctx, pgUUID(exchangeID))
		if err != nil {
			return fmt.Errorf("count incomplete exchange participants: %w", err)
		}
		if incomplete > 0 {
			return nil
		}

		return finalizeExchange(ctx, queries, exchangeID)
	})
	if err != nil {
		return fmt.Errorf("complete exchange participation: %w", err)
	}

	return nil
}

func finalizeExchange(
	ctx context.Context,
	queries exchangeWriteQueries,
	exchangeID uuid.UUID,
) error {
	items, err := queries.LockExchangeItems(ctx, pgUUID(exchangeID))
	if err != nil {
		return fmt.Errorf("lock exchange items for completion: %w", err)
	}
	if len(items) == 0 {
		return ErrConflict
	}
	for _, item := range items {
		if item.Status != db.ItemStatusReserved {
			return ErrConflict
		}
	}

	traded, err := queries.MarkExchangeItemsTraded(ctx, pgUUID(exchangeID))
	if err != nil {
		return fmt.Errorf("mark exchange items traded: %w", err)
	}
	if traded != int64(len(items)) {
		return ErrConflict
	}

	if err := queries.CompleteExchange(ctx, pgUUID(exchangeID)); err != nil {
		return fmt.Errorf("complete exchange: %w", err)
	}

	updatedUsers, err := queries.IncrementExchangeParticipantsDealsCompleted(ctx, pgUUID(exchangeID))
	if err != nil {
		return fmt.Errorf("increment participant completed exchanges: %w", err)
	}
	if updatedUsers != int64(len(items)) {
		return ErrConflict
	}

	return recordExchangeEvent(
		ctx,
		queries,
		exchangeID,
		pgtype.UUID{},
		db.ChainMessageKindExchangeCompleted,
	)
}
