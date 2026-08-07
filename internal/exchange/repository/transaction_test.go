package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestWithinTransactionCommits(t *testing.T) {
	t.Parallel()

	tx := &fakeTx{}
	manager := &pgxTransactionManager{pool: &fakeTransactionBeginner{tx: tx}}

	err := manager.WithinTransaction(context.Background(), func(exchangeWriteQueries) error {
		return nil
	})
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}

	if !tx.committed {
		t.Fatal("transaction was not committed")
	}

	if tx.rolledBack {
		t.Fatal("committed transaction was rolled back")
	}
}

func TestWithinTransactionRollsBackOperationError(t *testing.T) {
	t.Parallel()

	operationError := errors.New("operation failed")
	tx := &fakeTx{}
	manager := &pgxTransactionManager{pool: &fakeTransactionBeginner{tx: tx}}

	err := manager.WithinTransaction(context.Background(), func(exchangeWriteQueries) error {
		return operationError
	})
	if !errors.Is(err, operationError) {
		t.Fatalf("WithinTransaction() error = %v, want %v", err, operationError)
	}

	if tx.committed {
		t.Fatal("failed transaction was committed")
	}

	if !tx.rolledBack {
		t.Fatal("failed transaction was not rolled back")
	}
}

func TestWithinTransactionBeginError(t *testing.T) {
	t.Parallel()

	beginError := errors.New("begin failed")
	manager := &pgxTransactionManager{
		pool: &fakeTransactionBeginner{err: beginError},
	}

	err := manager.WithinTransaction(context.Background(), func(exchangeWriteQueries) error {
		t.Fatal("operation must not be called")
		return nil
	})
	if !errors.Is(err, beginError) {
		t.Fatalf("WithinTransaction() error = %v, want wrapped %v", err, beginError)
	}
}

func TestWithinTransactionCommitErrorRollsBack(t *testing.T) {
	t.Parallel()

	commitError := errors.New("commit failed")
	tx := &fakeTx{commitErr: commitError}
	manager := &pgxTransactionManager{pool: &fakeTransactionBeginner{tx: tx}}

	err := manager.WithinTransaction(context.Background(), func(exchangeWriteQueries) error {
		return nil
	})
	if !errors.Is(err, commitError) {
		t.Fatalf("WithinTransaction() error = %v, want wrapped %v", err, commitError)
	}

	if !tx.rolledBack {
		t.Fatal("transaction was not rolled back after commit error")
	}
}

type fakeTransactionBeginner struct {
	tx  pgx.Tx
	err error
}

func (f *fakeTransactionBeginner) Begin(context.Context) (pgx.Tx, error) {
	return f.tx, f.err
}

type fakeTx struct {
	pgx.Tx
	commitErr  error
	committed  bool
	rolledBack bool
}

func (f *fakeTx) Commit(context.Context) error {
	f.committed = true
	return f.commitErr
}

func (f *fakeTx) Rollback(context.Context) error {
	f.rolledBack = true
	return nil
}
