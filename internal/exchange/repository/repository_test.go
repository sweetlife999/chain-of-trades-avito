package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

func TestFindNeighbors(t *testing.T) {
	t.Parallel()

	itemID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000000")
	firstItemID := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000000")
	firstOwnerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	secondItemID := uuid.MustParse("cccccccc-0000-0000-0000-000000000000")
	secondOwnerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	queries := &fakeNeighborQueries{
		rows: []db.FindExchangeNeighborsRow{
			{ID: pgUUID(firstItemID), OwnerID: pgUUID(firstOwnerID)},
			{ID: pgUUID(secondItemID), OwnerID: pgUUID(secondOwnerID)},
		},
	}

	repository := New(queries)
	neighbors, err := repository.FindNeighbors(context.Background(), itemID)
	if err != nil {
		t.Fatalf("FindNeighbors() error = %v", err)
	}

	if queries.receivedItemID != pgUUID(itemID) {
		t.Fatalf("query item ID = %v, want %v", queries.receivedItemID, pgUUID(itemID))
	}

	if len(neighbors) != 2 {
		t.Fatalf("neighbors count = %d, want 2", len(neighbors))
	}

	if neighbors[0].ItemID != firstItemID || neighbors[0].OwnerID != firstOwnerID {
		t.Errorf("first neighbor = %+v, want item %s owned by %s", neighbors[0], firstItemID, firstOwnerID)
	}

	if neighbors[1].ItemID != secondItemID || neighbors[1].OwnerID != secondOwnerID {
		t.Errorf("second neighbor = %+v, want item %s owned by %s", neighbors[1], secondItemID, secondOwnerID)
	}
}

func TestFindNeighborsEmpty(t *testing.T) {
	t.Parallel()

	repository := New(&fakeNeighborQueries{})
	neighbors, err := repository.FindNeighbors(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("FindNeighbors() error = %v", err)
	}

	if neighbors == nil {
		t.Fatal("FindNeighbors() returned nil, want an empty slice")
	}

	if len(neighbors) != 0 {
		t.Fatalf("neighbors count = %d, want 0", len(neighbors))
	}
}

func TestFindNeighborsError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := New(&fakeNeighborQueries{err: databaseError})

	_, err := repository.FindNeighbors(context.Background(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("FindNeighbors() error = %v, want wrapped %v", err, databaseError)
	}
}

type fakeNeighborQueries struct {
	rows           []db.FindExchangeNeighborsRow
	err            error
	receivedItemID pgtype.UUID
}

func (f *fakeNeighborQueries) FindExchangeNeighbors(
	_ context.Context,
	itemID pgtype.UUID,
) ([]db.FindExchangeNeighborsRow, error) {
	f.receivedItemID = itemID
	return f.rows, f.err
}
