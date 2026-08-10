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

// Вещи могли уехать в пункты ещё до сборки обмена: тогда подтверждение — единственный
// момент, когда цепочка может уйти в доставку, и сдавать после него уже нечего.
func TestConfirmParticipationMovesPreDeliveredExchangeToDelivering(t *testing.T) {
	t.Parallel()

	queries := pendingDecisionQueries()
	queries.items = []db.LockExchangeItemsRow{
		{ID: pgUUID(uuid.New()), Status: db.ItemStatusAvailable},
		{ID: pgUUID(uuid.New()), Status: db.ItemStatusAvailable},
	}
	queries.reserved = 2
	queries.promoted = 1
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	if err := repository.ConfirmParticipation(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("ConfirmParticipation() error = %v", err)
	}

	assertRecordedEvents(
		t,
		queries,
		db.ChainMessageKindParticipantAccepted,
		db.ChainMessageKindExchangeConfirmed,
		db.ChainMessageKindExchangeDelivering,
	)
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
	itemID := uuid.New()
	ownerID := uuid.New()
	queries.items = []db.LockExchangeItemsRow{{
		ID:      pgUUID(itemID),
		OwnerID: pgUUID(ownerID),
		Status:  db.ItemStatusAvailable,
	}}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	exchangeID := uuid.New()
	userID := uuid.New()

	nodes, composition, err := repository.DeclineParticipation(context.Background(), exchangeID, userID)
	if err != nil {
		t.Fatalf("DeclineParticipation() error = %v", err)
	}
	// Отказ от предложения ничего не освобождает, но перепоиск от его объявлений всё
	// равно нужен: другого канала обнаружения обмена у участников нет.
	if len(nodes) != 1 || nodes[0].ItemID != itemID || nodes[0].OwnerID != ownerID {
		t.Fatalf("recovery nodes = %+v, want item %s owned by %s", nodes, itemID, ownerID)
	}
	if composition != queries.chainComposition {
		t.Fatalf("cancelled composition = %q, want %q", composition, queries.chainComposition)
	}
	wantRefusal := db.RecordItemRefusalParams{ExchangeID: pgUUID(exchangeID), UserID: pgUUID(userID)}
	if len(queries.refusals) != 1 || queries.refusals[0] != wantRefusal {
		t.Fatalf("recorded refusals = %+v, want exactly %+v", queries.refusals, wantRefusal)
	}
	if !queries.declined || !queries.cancelled {
		t.Fatalf(
			"declined = %v, cancelled = %v; want both true",
			queries.declined,
			queries.cancelled,
		)
	}
	if queries.cancelParams.CancelReason != db.ChainCancelReasonProposalDeclined {
		t.Fatalf(
			"cancel reason = %q, want %q",
			queries.cancelParams.CancelReason,
			db.ChainCancelReasonProposalDeclined,
		)
	}
	if queries.brokenCompositionCalled {
		t.Fatal("proposed decline permanently blocked the whole composition")
	}
	if !transactions.committed {
		t.Fatal("decline transaction was not committed")
	}
	assertRecordedEvents(t, queries, db.ChainMessageKindParticipantDeclined)
}

func TestDeclineConfirmedExchangeReleasesItems(t *testing.T) {
	t.Parallel()

	itemID := uuid.New()
	ownerID := uuid.New()
	queries := pendingDecisionQueries()
	queries.chainStatus = db.ChainStatusConfirmed
	queries.participantStatus = db.ParticipantStatusAccepted
	queries.items = []db.LockExchangeItemsRow{{
		ID:      pgUUID(itemID),
		OwnerID: pgUUID(ownerID),
		Status:  db.ItemStatusReserved,
	}}
	queries.released = 1
	queries.dealsBrokenUpdated = 1
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	nodes, composition, err := repository.DeclineParticipation(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("DeclineParticipation() error = %v", err)
	}
	if len(nodes) != 1 || nodes[0].ItemID != itemID || nodes[0].OwnerID != ownerID {
		t.Fatalf("recovery nodes = %+v, want item %s owned by %s", nodes, itemID, ownerID)
	}
	if composition != queries.chainComposition {
		t.Fatalf("cancelled composition = %q, want %q", composition, queries.chainComposition)
	}
	// Сорванная сделка — не «не хочу эту вещь», ребро графа остаётся на месте.
	if len(queries.refusals) != 0 {
		t.Fatalf("recorded refusals = %+v, want none for a broken confirmed exchange", queries.refusals)
	}
	if !queries.declined || !queries.cancelled || !queries.releaseCalled || !queries.dealsBrokenCalled {
		t.Fatalf(
			"declined=%v cancelled=%v releaseCalled=%v dealsBrokenCalled=%v",
			queries.declined,
			queries.cancelled,
			queries.releaseCalled,
			queries.dealsBrokenCalled,
		)
	}
	if !queries.brokenCompositionCalled {
		t.Fatal("broken confirmed composition was not persisted")
	}
	if queries.cancelParams.CancelReason != db.ChainCancelReasonConfirmedBroken {
		t.Fatalf(
			"cancel reason = %q, want %q",
			queries.cancelParams.CancelReason,
			db.ChainCancelReasonConfirmedBroken,
		)
	}
	if !transactions.committed {
		t.Fatal("confirmed decline transaction was not committed")
	}
}

func TestDeclineConfirmedExchangeRollsBackWhenCompositionCannotBePersisted(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("persist broken composition failed")
	queries := pendingDecisionQueries()
	queries.chainStatus = db.ChainStatusConfirmed
	queries.participantStatus = db.ParticipantStatusAccepted
	queries.items = []db.LockExchangeItemsRow{{Status: db.ItemStatusReserved}}
	queries.brokenCompositionErr = databaseError
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	_, _, err := repository.DeclineParticipation(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("DeclineParticipation() error = %v, want wrapped %v", err, databaseError)
	}
	if queries.releaseCalled || queries.cancelled {
		t.Fatal("exchange was changed without persisting the permanent exclusion")
	}
	if transactions.committed {
		t.Fatal("failed confirmed decline transaction was committed")
	}
}

func TestDeclineConfirmedExchangeAfterCompletionConfirmationConflicts(t *testing.T) {
	t.Parallel()

	queries := pendingDecisionQueries()
	queries.chainStatus = db.ChainStatusConfirmed
	queries.participantStatus = db.ParticipantStatusAccepted
	queries.participantCompletedAt.Valid = true
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	_, _, err := repository.DeclineParticipation(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("DeclineParticipation() error = %v, want %v", err, ErrConflict)
	}
	if transactions.committed {
		t.Fatal("conflicting decline transaction was committed")
	}
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
		chainSignature:    "item-a>item-b|item-b>item-a",
		chainComposition:  "item-a|item-b",
		participantStatus: db.ParticipantStatusPending,
	}
}
