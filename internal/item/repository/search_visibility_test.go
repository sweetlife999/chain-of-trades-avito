package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

func TestSetSearchVisibilityWithdrawsAndReturnsRecoveryCandidates(t *testing.T) {
	t.Parallel()

	itemID, ownerID := uuid.New(), uuid.New()
	otherItemID, otherOwnerID := uuid.New(), uuid.New()
	queries := visibilityFake(itemID, ownerID, db.ItemStatusAvailable)
	queries.recoveryRows = []db.CancelProposedExchangesForItemWithdrawalRow{
		{ItemID: pgUUID(otherItemID), OwnerID: pgUUID(otherOwnerID)},
	}
	repository := repositoryWithVisibilityQueries(queries)

	result, err := repository.SetSearchVisibility(context.Background(), itemID, ownerID, false)
	if err != nil {
		t.Fatalf("SetSearchVisibility() error = %v", err)
	}
	if !result.Changed || result.Item.Status != string(db.ItemStatusWithdrawn) {
		t.Fatalf("change = %+v, want changed withdrawn item", result)
	}
	if queries.withdrawCalls != 1 || queries.cancelCalls != 1 {
		t.Fatalf("withdraw calls = %d, cancel calls = %d, want 1 and 1", queries.withdrawCalls, queries.cancelCalls)
	}
	if len(result.RecoveryCandidates) != 1 ||
		result.RecoveryCandidates[0].ItemID != otherItemID ||
		result.RecoveryCandidates[0].OwnerID != otherOwnerID {
		t.Fatalf("recovery candidates = %+v", result.RecoveryCandidates)
	}
}

func TestSetSearchVisibilityPublishesWithdrawnItem(t *testing.T) {
	t.Parallel()

	itemID, ownerID := uuid.New(), uuid.New()
	queries := visibilityFake(itemID, ownerID, db.ItemStatusWithdrawn)
	repository := repositoryWithVisibilityQueries(queries)

	result, err := repository.SetSearchVisibility(context.Background(), itemID, ownerID, true)
	if err != nil {
		t.Fatalf("SetSearchVisibility() error = %v", err)
	}
	if !result.Changed || result.Item.Status != string(db.ItemStatusAvailable) {
		t.Fatalf("change = %+v, want changed available item", result)
	}
	if queries.publishCalls != 1 || queries.cancelCalls != 0 {
		t.Fatalf("publish calls = %d, cancel calls = %d, want 1 and 0", queries.publishCalls, queries.cancelCalls)
	}
}

func TestSetSearchVisibilityIsIdempotent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  db.ItemStatus
		enabled bool
	}{
		{name: "already available", status: db.ItemStatusAvailable, enabled: true},
		{name: "already withdrawn", status: db.ItemStatusWithdrawn, enabled: false},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			itemID, ownerID := uuid.New(), uuid.New()
			queries := visibilityFake(itemID, ownerID, test.status)
			result, err := repositoryWithVisibilityQueries(queries).SetSearchVisibility(
				context.Background(), itemID, ownerID, test.enabled,
			)
			if err != nil {
				t.Fatalf("SetSearchVisibility() error = %v", err)
			}
			if result.Changed || queries.withdrawCalls != 0 || queries.publishCalls != 0 || queries.cancelCalls != 0 {
				t.Fatalf("idempotent change = %+v, mutations = (%d, %d, %d)",
					result, queries.withdrawCalls, queries.publishCalls, queries.cancelCalls)
			}
		})
	}
}

func TestSetSearchVisibilityRejectsWrongOwner(t *testing.T) {
	t.Parallel()

	itemID := uuid.New()
	queries := visibilityFake(itemID, uuid.New(), db.ItemStatusAvailable)
	_, err := repositoryWithVisibilityQueries(queries).SetSearchVisibility(
		context.Background(), itemID, uuid.New(), false,
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("SetSearchVisibility() error = %v, want %v", err, ErrForbidden)
	}
	if queries.withdrawCalls != 0 || queries.cancelCalls != 0 {
		t.Fatal("wrong owner changed item or exchanges")
	}
}

func TestSetSearchVisibilityRejectsOccupiedOrTradedItem(t *testing.T) {
	t.Parallel()

	for _, status := range []db.ItemStatus{db.ItemStatusReserved, db.ItemStatusTraded} {
		status := status
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			itemID, ownerID := uuid.New(), uuid.New()
			queries := visibilityFake(itemID, ownerID, status)
			_, err := repositoryWithVisibilityQueries(queries).SetSearchVisibility(
				context.Background(), itemID, ownerID, false,
			)
			if !errors.Is(err, ErrSearchVisibilityConflict) {
				t.Fatalf("SetSearchVisibility() error = %v, want %v", err, ErrSearchVisibilityConflict)
			}
		})
	}
}

func TestSetSearchVisibilityRollsBackCancellationError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	itemID, ownerID := uuid.New(), uuid.New()
	queries := visibilityFake(itemID, ownerID, db.ItemStatusAvailable)
	queries.cancelErr = databaseError
	manager := &fakeVisibilityTransactions{queries: queries}
	repository := &Repository{transactions: manager}

	_, err := repository.SetSearchVisibility(context.Background(), itemID, ownerID, false)
	if !errors.Is(err, databaseError) {
		t.Fatalf("SetSearchVisibility() error = %v, want wrapped %v", err, databaseError)
	}
	if manager.committed {
		t.Fatal("transaction committed after cancellation error")
	}
}

type fakeVisibilityTransactions struct {
	queries   *fakeVisibilityQueries
	committed bool
}

func (f *fakeVisibilityTransactions) WithinTransaction(
	ctx context.Context,
	operation func(searchVisibilityQueries) error,
) error {
	if err := operation(f.queries); err != nil {
		return err
	}
	f.committed = true
	return nil
}

type fakeVisibilityQueries struct {
	state         db.GetItemSearchVisibilityForUpdateRow
	item          db.GetItemByIDRow
	recoveryRows  []db.CancelProposedExchangesForItemWithdrawalRow
	lockErr       error
	stateErr      error
	withdrawErr   error
	publishErr    error
	cancelErr     error
	readErr       error
	withdrawCalls int
	publishCalls  int
	cancelCalls   int
}

func visibilityFake(itemID, ownerID uuid.UUID, status db.ItemStatus) *fakeVisibilityQueries {
	return &fakeVisibilityQueries{
		state: db.GetItemSearchVisibilityForUpdateRow{OwnerID: pgUUID(ownerID), Status: status},
		item: db.GetItemByIDRow{
			ID:        pgUUID(itemID),
			OwnerID:   pgUUID(ownerID),
			Status:    status,
			PhotoUrls: []string{"https://example.com/item.jpg"},
		},
	}
}

func repositoryWithVisibilityQueries(queries *fakeVisibilityQueries) *Repository {
	return &Repository{transactions: &fakeVisibilityTransactions{queries: queries}}
}

func (f *fakeVisibilityQueries) LockItemSearchVisibility(context.Context, pgtype.UUID) error {
	return f.lockErr
}

func (f *fakeVisibilityQueries) GetItemSearchVisibilityForUpdate(
	context.Context,
	pgtype.UUID,
) (db.GetItemSearchVisibilityForUpdateRow, error) {
	return f.state, f.stateErr
}

func (f *fakeVisibilityQueries) WithdrawAvailableItem(
	_ context.Context,
	_ db.WithdrawAvailableItemParams,
) (int64, error) {
	f.withdrawCalls++
	if f.withdrawErr == nil {
		f.item.Status = db.ItemStatusWithdrawn
	}
	return 1, f.withdrawErr
}

func (f *fakeVisibilityQueries) PublishWithdrawnItem(
	_ context.Context,
	_ db.PublishWithdrawnItemParams,
) (int64, error) {
	f.publishCalls++
	if f.publishErr == nil {
		f.item.Status = db.ItemStatusAvailable
	}
	return 1, f.publishErr
}

func (f *fakeVisibilityQueries) CancelProposedExchangesForItemWithdrawal(
	context.Context,
	pgtype.UUID,
) ([]db.CancelProposedExchangesForItemWithdrawalRow, error) {
	f.cancelCalls++
	return f.recoveryRows, f.cancelErr
}

func (f *fakeVisibilityQueries) GetItemByID(context.Context, pgtype.UUID) (db.GetItemByIDRow, error) {
	return f.item, f.readErr
}
