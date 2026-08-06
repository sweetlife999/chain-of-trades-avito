package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

func TestCompleteParticipationWaitsForOtherParticipants(t *testing.T) {
	t.Parallel()

	queries := completionQueries()
	queries.incomplete = 1
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	if err := repository.CompleteParticipation(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("CompleteParticipation() error = %v", err)
	}
	if !queries.completionConfirmed {
		t.Fatal("participant completion was not saved")
	}
	if queries.completed || queries.dealsCompletedUpdated != 0 {
		t.Fatal("exchange was finalized before all participants confirmed completion")
	}
	if !transactions.committed {
		t.Fatal("successful completion confirmation was not committed")
	}
}

func TestCompleteParticipationFinalizesExchange(t *testing.T) {
	t.Parallel()

	queries := completionQueries()
	queries.items = []db.LockExchangeItemsRow{
		{Status: db.ItemStatusReserved},
		{Status: db.ItemStatusReserved},
		{Status: db.ItemStatusReserved},
	}
	queries.traded = 3
	queries.dealsCompletedUpdated = 3
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	if err := repository.CompleteParticipation(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("CompleteParticipation() error = %v", err)
	}
	if !queries.completed {
		t.Fatal("exchange was not marked completed")
	}
	if !transactions.committed {
		t.Fatal("finalized exchange was not committed")
	}
}

func TestCompleteParticipationIsIdempotent(t *testing.T) {
	t.Parallel()

	queries := completionQueries()
	queries.chainStatus = db.ChainStatusCompleted
	queries.completionParticipant.CompletionConfirmedAt = pgtype.Timestamptz{
		Time:  time.Now(),
		Valid: true,
	}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	if err := repository.CompleteParticipation(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("repeated CompleteParticipation() error = %v", err)
	}
	if queries.completionConfirmed || queries.completed || queries.dealsCompletedUpdated != 0 {
		t.Fatal("repeated request changed exchange data")
	}
	if !transactions.committed {
		t.Fatal("idempotent request was not committed")
	}
}

func TestCompleteParticipationRejectsInvalidState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(*fakeExchangeWriteQueries)
		wantErr error
	}{
		{
			name: "exchange not found",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.lockExchangeErr = pgx.ErrNoRows
			},
			wantErr: ErrNotFound,
		},
		{
			name: "not a participant",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.lockCompletionParticipantErr = pgx.ErrNoRows
			},
			wantErr: ErrNotParticipant,
		},
		{
			name: "exchange is proposed",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.chainStatus = db.ChainStatusProposed
			},
			wantErr: ErrConflict,
		},
		{
			name: "item is not reserved",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.items = []db.LockExchangeItemsRow{{Status: db.ItemStatusAvailable}}
			},
			wantErr: ErrConflict,
		},
		{
			name: "not all items were updated",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.items = []db.LockExchangeItemsRow{{Status: db.ItemStatusReserved}}
				queries.traded = 0
			},
			wantErr: ErrConflict,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			queries := completionQueries()
			test.prepare(queries)
			transactions := &fakeTransactionManager{queries: queries}
			repository := newRepository(&fakeNeighborQueries{}, transactions)

			err := repository.CompleteParticipation(context.Background(), uuid.New(), uuid.New())
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("CompleteParticipation() error = %v, want %v", err, test.wantErr)
			}
			if transactions.committed {
				t.Fatal("failed completion was committed")
			}
		})
	}
}

func completionQueries() *fakeExchangeWriteQueries {
	return &fakeExchangeWriteQueries{
		chainStatus: db.ChainStatusConfirmed,
		completionParticipant: db.LockExchangeCompletionParticipantRow{
			Status: db.ParticipantStatusAccepted,
		},
	}
}
