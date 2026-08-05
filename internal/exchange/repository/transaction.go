package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

type exchangeWriteQueries interface {
	CreateExchange(context.Context) (pgtype.UUID, error)
	CreateExchangeParticipant(context.Context, db.CreateExchangeParticipantParams) error
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
