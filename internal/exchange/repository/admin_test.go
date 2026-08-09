package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

func TestAdminMarksExchangeDelivered(t *testing.T) {
	t.Parallel()

	queries := pendingDecisionQueries()
	queries.chainStatus = db.ChainStatusDelivering
	queries.delivered = 1
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)
	exchangeID := uuid.New()
	adminID := uuid.New()

	if err := repository.MarkDeliveredByAdmin(context.Background(), exchangeID, adminID); err != nil {
		t.Fatalf("MarkDeliveredByAdmin() error = %v", err)
	}
	if !queries.decisionItemsLocked || !queries.deliverCalled || queries.delivered != 1 || !transactions.committed {
		t.Fatalf(
			"locked=%v deliverCalled=%v delivered=%d committed=%v",
			queries.decisionItemsLocked,
			queries.deliverCalled,
			queries.delivered,
			transactions.committed,
		)
	}
	if len(queries.systemMessages) != 1 ||
		queries.systemMessages[0].Kind != db.ChainMessageKindExchangeDelivered ||
		queries.systemMessages[0].AuthorID.Valid {
		t.Fatalf("system messages = %+v, want one authorless exchange_delivered", queries.systemMessages)
	}
	if len(queries.adminAudit) != 1 {
		t.Fatalf("admin audit entries = %d, want 1", len(queries.adminAudit))
	}
	audit := queries.adminAudit[0]
	if uuid.UUID(audit.AdminID.Bytes) != adminID ||
		audit.Action != db.AdminAuditActionExchangeDelivered ||
		audit.TargetType != db.AdminAuditTargetExchange ||
		uuid.UUID(audit.TargetID.Bytes) != exchangeID {
		t.Fatalf("admin audit = %+v", audit)
	}
	var metadata map[string]string
	if err := json.Unmarshal(audit.Metadata, &metadata); err != nil {
		t.Fatalf("decode audit metadata: %v", err)
	}
	if metadata["previous_status"] != "delivering" || metadata["new_status"] != "delivered" {
		t.Fatalf("audit metadata = %+v", metadata)
	}
}

func TestAdminMarkDeliveredIsIdempotent(t *testing.T) {
	t.Parallel()

	queries := pendingDecisionQueries()
	queries.chainStatus = db.ChainStatusDelivered
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	if err := repository.MarkDeliveredByAdmin(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("MarkDeliveredByAdmin() repeated call error = %v", err)
	}
	if len(queries.systemMessages) != 0 || len(queries.adminAudit) != 0 {
		t.Fatal("repeated delivery created another event or audit entry")
	}
	if queries.deliverCalled {
		t.Fatal("repeated delivery executed another status update")
	}
	if !transactions.committed {
		t.Fatal("idempotent admin delivery transaction was not committed")
	}
}

func TestAdminMarkDeliveredErrors(t *testing.T) {
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
			name: "wrong status",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.chainStatus = db.ChainStatusConfirmed
			},
			want: ErrConflict,
		},
		{
			name: "incomplete pickup data",
			prepare: func(queries *fakeExchangeWriteQueries) {
				queries.chainStatus = db.ChainStatusDelivering
				queries.delivered = 0
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

			err := repository.MarkDeliveredByAdmin(context.Background(), uuid.New(), uuid.New())
			if !errors.Is(err, test.want) {
				t.Fatalf("MarkDeliveredByAdmin() error = %v, want %v", err, test.want)
			}
			if transactions.committed {
				t.Fatal("failed admin delivery transaction was committed")
			}
		})
	}
}

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
