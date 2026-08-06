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

var (
	ErrNotParticipant = errors.New("user is not an exchange participant")
	ErrConflict       = errors.New("exchange decision conflicts with current state")
)

func (r *Repository) ConfirmParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	return r.decideParticipation(ctx, exchangeID, userID, db.ParticipantStatusAccepted)
}

func (r *Repository) DeclineParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	return r.decideParticipation(ctx, exchangeID, userID, db.ParticipantStatusDeclined)
}

func (r *Repository) decideParticipation(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
	decision db.ParticipantStatus,
) error {
	err := r.transactions.WithinTransaction(ctx, func(queries exchangeWriteQueries) error {
		if err := queries.LockExchangeDecisionItems(ctx, pgUUID(exchangeID)); err != nil {
			return fmt.Errorf("lock exchange decision items: %w", err)
		}

		status, err := queries.LockExchange(ctx, pgUUID(exchangeID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("lock exchange: %w", err)
		}
		if status != db.ChainStatusProposed {
			return ErrConflict
		}

		participantParams := db.LockExchangeParticipantParams{
			ExchangeID: pgUUID(exchangeID),
			UserID:     pgUUID(userID),
		}
		participantStatus, err := queries.LockExchangeParticipant(ctx, participantParams)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotParticipant
		}
		if err != nil {
			return fmt.Errorf("lock exchange participant: %w", err)
		}
		if participantStatus != db.ParticipantStatusPending {
			return ErrConflict
		}

		switch decision {
		case db.ParticipantStatusAccepted:
			return confirmParticipation(ctx, queries, exchangeID, userID)
		case db.ParticipantStatusDeclined:
			return declineParticipation(ctx, queries, exchangeID, userID)
		default:
			return fmt.Errorf("unsupported participant decision %q", decision)
		}
	})
	if err != nil {
		return fmt.Errorf("decide exchange participation: %w", err)
	}

	return nil
}

func confirmParticipation(
	ctx context.Context,
	queries exchangeWriteQueries,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	err := queries.AcceptExchangeParticipant(ctx, db.AcceptExchangeParticipantParams{
		ExchangeID: pgUUID(exchangeID),
		UserID:     pgUUID(userID),
	})
	if err != nil {
		return fmt.Errorf("accept exchange participant: %w", err)
	}

	err = recordExchangeEvent(
		ctx,
		queries,
		exchangeID,
		pgUUID(userID),
		db.ChainMessageKindParticipantAccepted,
	)
	if err != nil {
		return err
	}

	pending, err := queries.CountPendingExchangeParticipants(ctx, pgUUID(exchangeID))
	if err != nil {
		return fmt.Errorf("count pending exchange participants: %w", err)
	}
	if pending > 0 {
		return nil
	}

	items, err := queries.LockExchangeItems(ctx, pgUUID(exchangeID))
	if err != nil {
		return fmt.Errorf("lock exchange items: %w", err)
	}
	if len(items) == 0 {
		return ErrConflict
	}
	for _, item := range items {
		if item.Status != db.ItemStatusAvailable {
			return ErrConflict
		}
	}

	reserved, err := queries.ReserveExchangeItems(ctx, pgUUID(exchangeID))
	if err != nil {
		return fmt.Errorf("reserve exchange items: %w", err)
	}
	if reserved != int64(len(items)) {
		return ErrConflict
	}

	if err := queries.ConfirmExchange(ctx, pgUUID(exchangeID)); err != nil {
		return fmt.Errorf("confirm exchange: %w", err)
	}

	err = recordExchangeEvent(
		ctx,
		queries,
		exchangeID,
		pgtype.UUID{},
		db.ChainMessageKindExchangeConfirmed,
	)
	if err != nil {
		return err
	}

	if _, err := queries.CancelCompetingProposedExchanges(ctx, pgUUID(exchangeID)); err != nil {
		return fmt.Errorf("cancel competing proposed exchanges: %w", err)
	}

	return nil
}

func declineParticipation(
	ctx context.Context,
	queries exchangeWriteQueries,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) error {
	err := queries.DeclineExchangeParticipant(ctx, db.DeclineExchangeParticipantParams{
		ExchangeID: pgUUID(exchangeID),
		UserID:     pgUUID(userID),
	})
	if err != nil {
		return fmt.Errorf("decline exchange participant: %w", err)
	}

	// Отдельное событие «обмен отменён» не пишем: отказ участника стоит прямо перед ним
	// и говорит и о причине, и о последствии.
	err = recordExchangeEvent(
		ctx,
		queries,
		exchangeID,
		pgUUID(userID),
		db.ChainMessageKindParticipantDeclined,
	)
	if err != nil {
		return err
	}

	if err := queries.CancelExchange(ctx, pgUUID(exchangeID)); err != nil {
		return fmt.Errorf("cancel exchange: %w", err)
	}

	return nil
}
