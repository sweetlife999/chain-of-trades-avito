package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

func TestConfirmParticipationWaitsForOtherParticipants(t *testing.T) {
	t.Parallel()

	queries := pendingDecisionQueries()
	queries.pending = 1
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	err := repository.ConfirmParticipation(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("ConfirmParticipation() error = %v", err)
	}
	if !queries.accepted {
		t.Fatal("participant was not accepted")
	}
	if !queries.decisionItemsLocked {
		t.Fatal("exchange items were not locked before the decision")
	}
	if queries.confirmed {
		t.Fatal("exchange was confirmed before all participants accepted")
	}
	if !transactions.committed {
		t.Fatal("decision transaction was not committed")
	}
	assertRecordedEvents(t, queries, db.ChainMessageKindParticipantAccepted)
}

func TestConfirmParticipationReservesItemsAfterLastAcceptance(t *testing.T) {
	t.Parallel()

	queries := pendingDecisionQueries()
	queries.items = []db.LockExchangeItemsRow{
		{ID: pgUUID(uuid.New()), Status: db.ItemStatusAvailable},
		{ID: pgUUID(uuid.New()), Status: db.ItemStatusAvailable},
	}
	queries.reserved = 2
	queries.competingCancelled = 1
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	userID := uuid.New()

	err := repository.ConfirmParticipation(context.Background(), uuid.New(), userID)
	if err != nil {
		t.Fatalf("ConfirmParticipation() error = %v", err)
	}
	if !queries.confirmed {
		t.Fatal("exchange was not confirmed")
	}
	if !queries.cancelCompetingCalled {
		t.Fatal("competing proposed exchanges were not cancelled")
	}
	if !transactions.committed {
		t.Fatal("decision transaction was not committed")
	}

	assertRecordedEvents(
		t,
		queries,
		db.ChainMessageKindParticipantAccepted,
		db.ChainMessageKindExchangeConfirmed,
	)
	if queries.systemMessages[0].AuthorID != pgUUID(userID) {
		t.Fatalf("acceptance author = %v, want %v", queries.systemMessages[0].AuthorID, pgUUID(userID))
	}
	if queries.systemMessages[1].AuthorID.Valid {
		t.Fatal("exchange confirmation must belong to the exchange, not to a participant")
	}
}

func TestConfirmParticipationRejectsUnavailableItem(t *testing.T) {
	t.Parallel()

	queries := pendingDecisionQueries()
	queries.items = []db.LockExchangeItemsRow{
		{ID: pgUUID(uuid.New()), Status: db.ItemStatusReserved},
	}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	err := repository.ConfirmParticipation(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("ConfirmParticipation() error = %v, want %v", err, ErrConflict)
	}
	if transactions.committed {
		t.Fatal("conflicting decision transaction was committed")
	}
}

func TestConfirmParticipationRollsBackWhenCompetitorsCannotBeCancelled(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("cancel competitors failed")
	queries := pendingDecisionQueries()
	queries.items = []db.LockExchangeItemsRow{
		{ID: pgUUID(uuid.New()), Status: db.ItemStatusAvailable},
	}
	queries.reserved = 1
	queries.cancelCompetingErr = databaseError
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	err := repository.ConfirmParticipation(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("ConfirmParticipation() error = %v, want wrapped %v", err, databaseError)
	}
	if transactions.committed {
		t.Fatal("transaction was committed without cancelling competing exchanges")
	}
}

func TestDeclineParticipationCancelsExchange(t *testing.T) {
	t.Parallel()

	queries := pendingDecisionQueries()
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	err := repository.DeclineParticipation(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("DeclineParticipation() error = %v", err)
	}
	if !queries.declined || !queries.cancelled {
		t.Fatalf(
			"declined = %v, cancelled = %v; want both true",
			queries.declined,
			queries.cancelled,
		)
	}
	if !transactions.committed {
		t.Fatal("decline transaction was not committed")
	}
	assertRecordedEvents(t, queries, db.ChainMessageKindParticipantDeclined)
}

func TestDecisionErrors(t *testing.T) {
	t.Parallel()

	lockError := errors.New("advisory lock failed")
	tests := []struct {
		name    string
		prepare func(*fakeExchangeWriteQueries)
		want    error
	}{
		{
			name: "cannot lock decision items",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.lockDecisionItemsErr = lockError
			},
			want: lockError,
		},
		{
			name: "exchange not found",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.lockExchangeErr = pgx.ErrNoRows
			},
			want: ErrNotFound,
		},
		{
			name: "not a participant",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.lockParticipantErr = pgx.ErrNoRows
			},
			want: ErrNotParticipant,
		},
		{
			name: "exchange already closed",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.chainStatus = db.ChainStatusConfirmed
			},
			want: ErrConflict,
		},
		{
			name: "participant already decided",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.participantStatus = db.ParticipantStatusAccepted
			},
			want: ErrConflict,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			queries := pendingDecisionQueries()
			test.prepare(queries)
			transactions := &fakeTransactionManager{queries: queries}
			repository := newRepository(&fakeNeighborQueries{}, transactions)

			err := repository.ConfirmParticipation(context.Background(), uuid.New(), uuid.New())
			if !errors.Is(err, test.want) {
				t.Fatalf("ConfirmParticipation() error = %v, want %v", err, test.want)
			}
			if transactions.committed {
				t.Fatal("failed decision transaction was committed")
			}
		})
	}
}

func pendingDecisionQueries() *fakeExchangeWriteQueries {
	return &fakeExchangeWriteQueries{
		chainStatus:       db.ChainStatusProposed,
		participantStatus: db.ParticipantStatusPending,
	}
}
