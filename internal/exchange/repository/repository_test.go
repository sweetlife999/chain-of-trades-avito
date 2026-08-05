package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
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

	repository := newRepository(queries, nil)
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

	repository := newRepository(&fakeNeighborQueries{}, nil)
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
	repository := newRepository(&fakeNeighborQueries{err: databaseError}, nil)

	_, err := repository.FindNeighbors(context.Background(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("FindNeighbors() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestSaveExchange(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	firstUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	secondUserID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	firstItemID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	secondItemID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	exchange := exchangemodel.Exchange{Participants: []exchangemodel.Participant{
		{
			UserID:         firstUserID,
			GivesItemID:    firstItemID,
			ReceivesItemID: secondItemID,
			Position:       0,
		},
		{
			UserID:         secondUserID,
			GivesItemID:    secondItemID,
			ReceivesItemID: firstItemID,
			Position:       1,
		},
	}}

	queries := &fakeExchangeWriteQueries{exchangeID: pgUUID(exchangeID)}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	actualID, err := repository.SaveExchange(context.Background(), exchange)
	if err != nil {
		t.Fatalf("SaveExchange() error = %v", err)
	}

	if actualID != exchangeID {
		t.Fatalf("SaveExchange() ID = %s, want %s", actualID, exchangeID)
	}

	wantParams := []db.CreateExchangeParticipantParams{
		{
			ChainID:        pgUUID(exchangeID),
			UserID:         pgUUID(firstUserID),
			GivesItemID:    pgUUID(firstItemID),
			ReceivesItemID: pgUUID(secondItemID),
			Position:       0,
		},
		{
			ChainID:        pgUUID(exchangeID),
			UserID:         pgUUID(secondUserID),
			GivesItemID:    pgUUID(secondItemID),
			ReceivesItemID: pgUUID(firstItemID),
			Position:       1,
		},
	}

	if !reflect.DeepEqual(queries.participants, wantParams) {
		t.Fatalf("participants = %+v, want %+v", queries.participants, wantParams)
	}

	if !transactions.committed {
		t.Fatal("transaction was not committed")
	}
}

func TestSaveExchangeCreateErrorRollsBack(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("create failed")
	queries := &fakeExchangeWriteQueries{createErr: databaseError}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	_, err := repository.SaveExchange(context.Background(), exchangemodel.Exchange{})
	if !errors.Is(err, databaseError) {
		t.Fatalf("SaveExchange() error = %v, want wrapped %v", err, databaseError)
	}

	if transactions.committed {
		t.Fatal("failed transaction was committed")
	}
}

func TestSaveExchangeParticipantErrorRollsBack(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("participant failed")
	queries := &fakeExchangeWriteQueries{
		exchangeID:     pgUUID(uuid.New()),
		participantErr: databaseError,
	}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	exchange := exchangemodel.Exchange{Participants: []exchangemodel.Participant{{
		UserID:         uuid.New(),
		GivesItemID:    uuid.New(),
		ReceivesItemID: uuid.New(),
	}}}

	_, err := repository.SaveExchange(context.Background(), exchange)
	if !errors.Is(err, databaseError) {
		t.Fatalf("SaveExchange() error = %v, want wrapped %v", err, databaseError)
	}

	if transactions.committed {
		t.Fatal("failed transaction was committed")
	}
}

func TestSaveExchangeTransactionError(t *testing.T) {
	t.Parallel()

	transactionError := errors.New("begin failed")
	repository := newRepository(
		&fakeNeighborQueries{},
		&fakeTransactionManager{err: transactionError},
	)

	_, err := repository.SaveExchange(context.Background(), exchangemodel.Exchange{})
	if !errors.Is(err, transactionError) {
		t.Fatalf("SaveExchange() error = %v, want wrapped %v", err, transactionError)
	}
}

type fakeNeighborQueries struct {
	rows           []db.FindExchangeNeighborsRow
	err            error
	receivedItemID pgtype.UUID
}

type fakeExchangeWriteQueries struct {
	exchangeID     pgtype.UUID
	createErr      error
	participantErr error
	participants   []db.CreateExchangeParticipantParams

	chainStatus        db.ChainStatus
	lockExchangeErr    error
	participantStatus  db.ParticipantStatus
	lockParticipantErr error
	accepted           bool
	acceptErr          error
	declined           bool
	declineErr         error
	pending            int64
	pendingErr         error
	items              []db.LockExchangeItemsRow
	lockItemsErr       error
	reserved           int64
	reserveErr         error
	confirmed          bool
	confirmErr         error
	cancelled          bool
	cancelErr          error
}

func (f *fakeExchangeWriteQueries) CreateExchange(context.Context) (pgtype.UUID, error) {
	return f.exchangeID, f.createErr
}

func (f *fakeExchangeWriteQueries) CreateExchangeParticipant(
	_ context.Context,
	params db.CreateExchangeParticipantParams,
) error {
	f.participants = append(f.participants, params)
	return f.participantErr
}

func (f *fakeExchangeWriteQueries) LockExchange(
	context.Context,
	pgtype.UUID,
) (db.ChainStatus, error) {
	return f.chainStatus, f.lockExchangeErr
}

func (f *fakeExchangeWriteQueries) LockExchangeParticipant(
	context.Context,
	db.LockExchangeParticipantParams,
) (db.ParticipantStatus, error) {
	return f.participantStatus, f.lockParticipantErr
}

func (f *fakeExchangeWriteQueries) AcceptExchangeParticipant(
	context.Context,
	db.AcceptExchangeParticipantParams,
) error {
	f.accepted = true
	return f.acceptErr
}

func (f *fakeExchangeWriteQueries) DeclineExchangeParticipant(
	context.Context,
	db.DeclineExchangeParticipantParams,
) error {
	f.declined = true
	return f.declineErr
}

func (f *fakeExchangeWriteQueries) CountPendingExchangeParticipants(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	return f.pending, f.pendingErr
}

func (f *fakeExchangeWriteQueries) LockExchangeItems(
	context.Context,
	pgtype.UUID,
) ([]db.LockExchangeItemsRow, error) {
	return f.items, f.lockItemsErr
}

func (f *fakeExchangeWriteQueries) ReserveExchangeItems(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	return f.reserved, f.reserveErr
}

func (f *fakeExchangeWriteQueries) ConfirmExchange(context.Context, pgtype.UUID) error {
	f.confirmed = true
	return f.confirmErr
}

func (f *fakeExchangeWriteQueries) CancelExchange(context.Context, pgtype.UUID) error {
	f.cancelled = true
	return f.cancelErr
}

type fakeTransactionManager struct {
	queries   exchangeWriteQueries
	err       error
	committed bool
}

func (f *fakeTransactionManager) WithinTransaction(
	_ context.Context,
	operation func(exchangeWriteQueries) error,
) error {
	if f.err != nil {
		return f.err
	}

	if err := operation(f.queries); err != nil {
		return err
	}

	f.committed = true
	return nil
}

func (f *fakeNeighborQueries) FindExchangeNeighbors(
	_ context.Context,
	itemID pgtype.UUID,
) ([]db.FindExchangeNeighborsRow, error) {
	f.receivedItemID = itemID
	return f.rows, f.err
}
