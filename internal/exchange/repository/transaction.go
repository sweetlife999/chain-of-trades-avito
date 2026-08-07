package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

type exchangeWriteQueries interface {
	CreateExchange(context.Context, string) (pgtype.UUID, error)
	CreateExchangeParticipant(context.Context, db.CreateExchangeParticipantParams) error
	LockExchangeDecisionItems(context.Context, pgtype.UUID) error
	LockExchange(context.Context, pgtype.UUID) (db.LockExchangeRow, error)
	LockExchangeParticipant(
		context.Context,
		db.LockExchangeParticipantParams,
	) (db.LockExchangeParticipantRow, error)
	AcceptExchangeParticipant(context.Context, db.AcceptExchangeParticipantParams) error
	DeclineExchangeParticipant(context.Context, db.DeclineExchangeParticipantParams) error
	CountPendingExchangeParticipants(context.Context, pgtype.UUID) (int64, error)
	LockExchangeItems(context.Context, pgtype.UUID) ([]db.LockExchangeItemsRow, error)
	ReserveExchangeItems(context.Context, pgtype.UUID) (int64, error)
	ConfirmExchange(context.Context, pgtype.UUID) error
	CancelCompetingProposedExchanges(context.Context, pgtype.UUID) (int64, error)
	CancelExchange(context.Context, pgtype.UUID) error
	CreateChainSystemMessage(context.Context, db.CreateChainSystemMessageParams) error
	ReleaseExchangeItems(context.Context, pgtype.UUID) (int64, error)
	IncrementUserDealsBroken(context.Context, pgtype.UUID) (int64, error)
	LockExchangeCompletionParticipant(
		context.Context,
		db.LockExchangeCompletionParticipantParams,
	) (db.LockExchangeCompletionParticipantRow, error)
	ConfirmExchangeParticipantCompletion(
		context.Context,
		db.ConfirmExchangeParticipantCompletionParams,
	) error
	CountIncompleteExchangeParticipants(context.Context, pgtype.UUID) (int64, error)
	MarkExchangeItemsTraded(context.Context, pgtype.UUID) (int64, error)
	CompleteExchange(context.Context, pgtype.UUID) error
	IncrementExchangeParticipantsDealsCompleted(context.Context, pgtype.UUID) (int64, error)
}

type transactionManager interface {
	WithinTransaction(context.Context, func(exchangeWriteQueries) error) error
}

type transactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

type pgxTransactionManager struct {
	pool transactionBeginner
}

func (m *pgxTransactionManager) WithinTransaction(
	ctx context.Context,
	operation func(exchangeWriteQueries) error,
) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := operation(db.New(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true

	return nil
}
