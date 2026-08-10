package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

func TestRecordItemPickupOutsideExchange(t *testing.T) {
	t.Parallel()

	queries := &fakeExchangeWriteQueries{pickupUpdated: 1}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	itemID, ownerID, pointID := uuid.New(), uuid.New(), uuid.New()

	if err := repository.RecordItemPickup(context.Background(), itemID, ownerID, pointID); err != nil {
		t.Fatalf("RecordItemPickup() error = %v", err)
	}
	if !queries.pickupLocked {
		t.Fatal("item was not locked before its pickup point was written")
	}
	if len(queries.pickupSet) != 1 || queries.pickupSet[0].PickupPointID != pgUUID(pointID) {
		t.Fatalf("pickup point writes = %v, want one for point %s", queries.pickupSet, pointID)
	}
	// Обмена нет — писать событие некуда, и переводить тоже нечего.
	assertRecordedEvents(t, queries)
	if !transactions.committed {
		t.Fatal("pickup transaction was not committed")
	}
}

func TestRecordItemPickupWaitsForOtherItems(t *testing.T) {
	t.Parallel()

	queries := confirmedPickupQueries()
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	if err := repository.RecordItemPickup(context.Background(), uuid.New(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("RecordItemPickup() error = %v", err)
	}
	// Соседи ещё не сдали свои вещи: запрос перевода вернул ноль строк.
	assertRecordedEvents(t, queries, db.ChainMessageKindParticipantDeliveredItem)
}

func TestRecordItemPickupMovesExchangeToDelivering(t *testing.T) {
	t.Parallel()

	queries := confirmedPickupQueries()
	queries.promoted = 1
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	ownerID := uuid.New()

	if err := repository.RecordItemPickup(context.Background(), uuid.New(), ownerID, uuid.New()); err != nil {
		t.Fatalf("RecordItemPickup() error = %v", err)
	}
	assertRecordedEvents(
		t,
		queries,
		db.ChainMessageKindParticipantDeliveredItem,
		db.ChainMessageKindExchangeDelivering,
	)
	if author := queries.systemMessages[0].AuthorID; author != pgUUID(ownerID) {
		t.Fatalf("pickup event author = %v, want %v", author, pgUUID(ownerID))
	}
	// Событие всей цепочки автора не имеет.
	if queries.systemMessages[1].AuthorID.Valid {
		t.Fatal("delivering event must not have an author")
	}
}

// Обмен мог уехать в доставку, пока запрос ждал блокировку: тогда вещь запоминается, но
// события уже не пишутся — иначе в треде появилась бы сдача после объявленной доставки.
func TestRecordItemPickupSkipsEventsWhenExchangeMovedOn(t *testing.T) {
	t.Parallel()

	queries := confirmedPickupQueries()
	queries.chainStatus = db.ChainStatusDelivering
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	if err := repository.RecordItemPickup(context.Background(), uuid.New(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("RecordItemPickup() error = %v", err)
	}
	if len(queries.pickupSet) != 1 {
		t.Fatalf("pickup point writes = %d, want 1", len(queries.pickupSet))
	}
	assertRecordedEvents(t, queries)
}

func TestRecordItemPickupFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(*fakeExchangeWriteQueries)
		wantErr error
	}{
		{
			name: "item is out of circulation",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.pickupUpdated = 0
			},
			wantErr: ErrConflict,
		},
		{
			name: "pickup point does not exist",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.setPickupErr = &pgconn.PgError{Code: "23503"}
			},
			wantErr: ErrUnknownPickupPoint,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			queries := confirmedPickupQueries()
			test.prepare(queries)
			transactions := &fakeTransactionManager{queries: queries}
			repository := newRepository(&fakeNeighborQueries{}, transactions)

			err := repository.RecordItemPickup(context.Background(), uuid.New(), uuid.New(), uuid.New())
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("RecordItemPickup() error = %v, want %v", err, test.wantErr)
			}
			if transactions.committed {
				t.Fatal("failed pickup was committed")
			}
		})
	}
}

func confirmedPickupQueries() *fakeExchangeWriteQueries {
	return &fakeExchangeWriteQueries{
		confirmedExchangeID: pgUUID(uuid.New()),
		chainStatus:         db.ChainStatusConfirmed,
		pickupUpdated:       1,
	}
}
