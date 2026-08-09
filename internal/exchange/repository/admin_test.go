package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

func TestAdminCancelProposedExchange(t *testing.T) {
	t.Parallel()

	itemID := uuid.New()
	ownerID := uuid.New()
	queries := pendingDecisionQueries()
	queries.items = []db.LockExchangeItemsRow{{
		ID:      pgUUID(itemID),
		OwnerID: pgUUID(ownerID),
		Status:  db.ItemStatusAvailable,
	}}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	adminID := uuid.New()
	nodes, signature, err := repository.CancelByAdmin(context.Background(), uuid.New(), adminID)
	if err != nil {
		t.Fatalf("CancelByAdmin() error = %v", err)
	}
	if len(nodes) != 1 || nodes[0].ItemID != itemID || nodes[0].OwnerID != ownerID {
		t.Fatalf("recovery nodes = %+v, want item %s owned by %s", nodes, itemID, ownerID)
	}
	if signature != queries.chainSignature {
		t.Fatalf("cancelled signature = %q, want %q", signature, queries.chainSignature)
	}
	if !queries.decisionItemsLocked || !queries.cancelled || queries.releaseCalled {
		t.Fatalf(
			"locked=%v cancelled=%v releaseCalled=%v",
			queries.decisionItemsLocked,
			queries.cancelled,
			queries.releaseCalled,
		)
	}
	if queries.declined || queries.dealsBrokenCalled || len(queries.refusals) != 0 {
		t.Fatal("admin cancellation changed participant decision or user statistics")
	}
	if !transactions.committed {
		t.Fatal("admin cancellation transaction was not committed")
	}
	if len(queries.adminAudit) != 1 || uuid.UUID(queries.adminAudit[0].AdminID.Bytes) != adminID {
		t.Fatalf("admin audit = %+v, want one entry by %s", queries.adminAudit, adminID)
	}
}

func TestAdminCancelConfirmedExchangeReleasesItems(t *testing.T) {
	t.Parallel()

	queries := pendingDecisionQueries()
	queries.chainStatus = db.ChainStatusConfirmed
	queries.items = []db.LockExchangeItemsRow{
		{ID: pgUUID(uuid.New()), OwnerID: pgUUID(uuid.New()), Status: db.ItemStatusReserved},
		{ID: pgUUID(uuid.New()), OwnerID: pgUUID(uuid.New()), Status: db.ItemStatusReserved},
	}
	queries.released = 2
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	nodes, _, err := repository.CancelByAdmin(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("CancelByAdmin() error = %v", err)
	}
	if len(nodes) != 2 || !queries.releaseCalled || !queries.cancelled {
		t.Fatalf("nodes=%d releaseCalled=%v cancelled=%v", len(nodes), queries.releaseCalled, queries.cancelled)
	}
	if queries.dealsBrokenCalled {
		t.Fatal("admin cancellation incremented a user's broken exchange counter")
	}
	if !transactions.committed {
		t.Fatal("admin cancellation transaction was not committed")
	}
}

func TestAdminCancelExchangeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(*fakeExchangeWriteQueries)
		want    error
	}{
		{
			name: "not found",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.lockExchangeErr = pgx.ErrNoRows
			},
			want: ErrNotFound,
		},
		{
			name: "already closed",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.chainStatus = db.ChainStatusCancelled
			},
			want: ErrConflict,
		},
		{
			name: "reserved item in proposed exchange",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.items = []db.LockExchangeItemsRow{{Status: db.ItemStatusReserved}}
			},
			want: ErrConflict,
		},
		{
			name: "partial release",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.chainStatus = db.ChainStatusConfirmed
				queries.items = []db.LockExchangeItemsRow{{Status: db.ItemStatusReserved}}
				queries.released = 0
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

			_, _, err := repository.CancelByAdmin(context.Background(), uuid.New(), uuid.New())
			if !errors.Is(err, test.want) {
				t.Fatalf("CancelByAdmin() error = %v, want %v", err, test.want)
			}
			if transactions.committed {
				t.Fatal("failed admin cancellation transaction was committed")
			}
		})
	}
}
