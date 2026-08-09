package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

func TestCancelByAdminRecoversWithoutRecreatingCancelledCycle(t *testing.T) {
	t.Parallel()

	nodes := makeNodes(3)
	repository := &fakeRepository{
		adminCancel: fakeAdminCancellation{
			recovery:    nodes,
			composition: compositionKey(nodes),
		},
		neighbors: cycleGraph(nodes),
	}
	exchangeID := uuid.New()

	if err := New(repository).CancelByAdmin(context.Background(), exchangeID, uuid.New()); err != nil {
		t.Fatalf("CancelByAdmin() error = %v", err)
	}
	if repository.adminCancel.exchangeID != exchangeID {
		t.Fatalf("CancelByAdmin() exchange ID = %s, want %s", repository.adminCancel.exchangeID, exchangeID)
	}
	if repository.saveCalls != 0 {
		t.Fatalf("SaveExchange() calls = %d, want cancelled cycle excluded", repository.saveCalls)
	}
}

func TestCancelByAdminRecoversWithAlternativeCycle(t *testing.T) {
	t.Parallel()

	original := makeNodes(3)
	alternative := testNode(4)
	repository := &fakeRepository{
		adminCancel: fakeAdminCancellation{
			recovery:    original,
			composition: compositionKey(original),
		},
		neighbors: map[uuid.UUID][]exchangemodel.Node{
			original[0].ItemID: {original[1]},
			original[1].ItemID: {original[2], alternative},
			original[2].ItemID: {original[0]},
			alternative.ItemID: {original[0]},
		},
	}

	if err := New(repository).CancelByAdmin(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("CancelByAdmin() error = %v", err)
	}
	if repository.saveCalls != 1 {
		t.Fatalf("SaveExchange() calls = %d, want one alternative", repository.saveCalls)
	}
}

func TestRecoveryStopsAfterFirstSavedExchange(t *testing.T) {
	t.Parallel()

	first := makeNodes(3)
	second := []exchangemodel.Node{testNode(4), testNode(5), testNode(6)}
	recovery := append(append([]exchangemodel.Node(nil), first...), second...)
	neighbors := cycleGraph(first)
	for itemID, candidates := range cycleGraph(second) {
		neighbors[itemID] = candidates
	}

	repository := &fakeRepository{
		adminCancel: fakeAdminCancellation{
			recovery:    recovery,
			composition: "cancelled-composition",
		},
		neighbors: neighbors,
	}

	if err := New(repository).CancelByAdmin(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("CancelByAdmin() error = %v", err)
	}
	if repository.saveCalls != 1 {
		t.Fatalf("SaveExchange() calls = %d, want recovery to stop after first success", repository.saveCalls)
	}
}

func TestCancelByAdminWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := &fakeRepository{adminCancel: fakeAdminCancellation{err: databaseError}}

	err := New(repository).CancelByAdmin(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("CancelByAdmin() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestMarkDeliveredByAdmin(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	exchangeID := uuid.New()
	adminID := uuid.New()
	if err := New(repository).MarkDeliveredByAdmin(context.Background(), exchangeID, adminID); err != nil {
		t.Fatalf("MarkDeliveredByAdmin() error = %v", err)
	}
	if repository.adminDelivery.exchangeID != exchangeID || repository.adminDelivery.adminID != adminID {
		t.Fatalf("admin delivery = %+v", repository.adminDelivery)
	}
}

func TestMarkDeliveredByAdminWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := &fakeRepository{adminDelivery: fakeAdminDelivery{err: databaseError}}
	err := New(repository).MarkDeliveredByAdmin(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("MarkDeliveredByAdmin() error = %v, want wrapped %v", err, databaseError)
	}
}
