package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func TestHasUserBlockConflict(t *testing.T) {
	t.Parallel()

	candidateID := uuid.New()
	pathIDs := []uuid.UUID{uuid.New(), uuid.New()}
	queries := &fakeNeighborQueries{blockConflict: true}
	repository := newRepository(queries, nil)

	conflict, err := repository.HasUserBlockConflict(context.Background(), candidateID, pathIDs)
	if err != nil {
		t.Fatalf("HasUserBlockConflict() error = %v", err)
	}
	if !conflict {
		t.Fatal("HasUserBlockConflict() = false, want true")
	}
	if queries.blockConflictParams.CandidateUserID != pgUUID(candidateID) {
		t.Fatalf("candidate ID = %v, want %v", queries.blockConflictParams.CandidateUserID, pgUUID(candidateID))
	}
	if len(queries.blockConflictParams.PathUserIds) != len(pathIDs) {
		t.Fatalf("path length = %d, want %d", len(queries.blockConflictParams.PathUserIds), len(pathIDs))
	}
	for index, id := range pathIDs {
		if queries.blockConflictParams.PathUserIds[index] != pgUUID(id) {
			t.Fatalf("path[%d] = %v, want %v", index, queries.blockConflictParams.PathUserIds[index], pgUUID(id))
		}
	}
}

func TestHasUserBlockConflictError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := newRepository(&fakeNeighborQueries{blockConflictErr: databaseError}, nil)

	_, err := repository.HasUserBlockConflict(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})
	if !errors.Is(err, databaseError) {
		t.Fatalf("HasUserBlockConflict() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestGetSearchUserStats(t *testing.T) {
	t.Parallel()

	firstID, secondID := uuid.New(), uuid.New()
	queries := &fakeNeighborQueries{statsRows: []db.ListExchangeSearchUserStatsRow{
		{ID: pgUUID(firstID), DealsCompleted: 12, DealsBroken: 1, Rating: 4.8},
		{ID: pgUUID(secondID), DealsCompleted: 2, DealsBroken: 3, Rating: 3.0},
	}}
	repository := newRepository(queries, nil)

	stats, err := repository.GetSearchUserStats(context.Background(), []uuid.UUID{firstID, secondID})
	if err != nil {
		t.Fatalf("GetSearchUserStats() error = %v", err)
	}
	if len(queries.statsUserIDs) != 2 || queries.statsUserIDs[0] != pgUUID(firstID) ||
		queries.statsUserIDs[1] != pgUUID(secondID) {
		t.Fatalf("query user IDs = %v", queries.statsUserIDs)
	}
	if stats[firstID].DealsCompleted != 12 || stats[firstID].DealsBroken != 1 || stats[firstID].Rating != 4.8 {
		t.Fatalf("first user stats = %+v", stats[firstID])
	}
	if stats[secondID].DealsCompleted != 2 || stats[secondID].DealsBroken != 3 || stats[secondID].Rating != 3.0 {
		t.Fatalf("second user stats = %+v", stats[secondID])
	}
}

func TestGetSearchUserStatsError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := newRepository(&fakeNeighborQueries{statsErr: databaseError}, nil)

	_, err := repository.GetSearchUserStats(context.Background(), []uuid.UUID{uuid.New()})
	if !errors.Is(err, databaseError) {
		t.Fatalf("GetSearchUserStats() error = %v, want wrapped %v", err, databaseError)
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

	queries := &fakeExchangeWriteQueries{
		exchangeID: pgUUID(exchangeID),
		items: []db.LockExchangeItemsRow{
			{ID: pgUUID(firstItemID), OwnerID: pgUUID(firstUserID), Status: db.ItemStatusAvailable},
			{ID: pgUUID(secondItemID), OwnerID: pgUUID(secondUserID), Status: db.ItemStatusAvailable},
		},
	}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	actualID, err := repository.SaveExchange(context.Background(), exchange)
	if err != nil {
		t.Fatalf("SaveExchange() error = %v", err)
	}

	if actualID != exchangeID {
		t.Fatalf("SaveExchange() ID = %s, want %s", actualID, exchangeID)
	}

	wantSignature := firstItemID.String() + ">" + secondItemID.String() + "|" +
		secondItemID.String() + ">" + firstItemID.String()
	if queries.signature != wantSignature {
		t.Fatalf("signature = %q, want %q", queries.signature, wantSignature)
	}
	wantComposition := firstItemID.String() + "|" + secondItemID.String()
	if queries.compositionKey != wantComposition {
		t.Fatalf("composition key = %q, want %q", queries.compositionKey, wantComposition)
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

func TestSaveExchangeDuplicateRollsBack(t *testing.T) {
	t.Parallel()

	queries := &fakeExchangeWriteQueries{createErr: pgx.ErrNoRows}
	transactions := &fakeTransactionManager{queries: queries}
	repository := newRepository(&fakeNeighborQueries{}, transactions)

	_, err := repository.SaveExchange(context.Background(), exchangemodel.Exchange{})
	if !errors.Is(err, ErrDuplicateExchange) {
		t.Fatalf("SaveExchange() error = %v, want %v", err, ErrDuplicateExchange)
	}
	if transactions.committed {
		t.Fatal("duplicate exchange transaction was committed")
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

func TestSaveExchangeRejectsStaleItems(t *testing.T) {
	t.Parallel()

	itemID := uuid.New()
	ownerID := uuid.New()
	exchange := exchangemodel.Exchange{Participants: []exchangemodel.Participant{{
		UserID:         ownerID,
		GivesItemID:    itemID,
		ReceivesItemID: itemID,
	}}}

	for _, test := range []struct {
		name  string
		items []db.LockExchangeItemsRow
	}{
		{name: "item disappeared"},
		{
			name: "item was reserved",
			items: []db.LockExchangeItemsRow{{
				ID:      pgUUID(itemID),
				OwnerID: pgUUID(ownerID),
				Status:  db.ItemStatusReserved,
			}},
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			queries := &fakeExchangeWriteQueries{exchangeID: pgUUID(uuid.New()), items: test.items}
			transactions := &fakeTransactionManager{queries: queries}
			repository := newRepository(&fakeNeighborQueries{}, transactions)

			_, err := repository.SaveExchange(context.Background(), exchange)
			if !errors.Is(err, ErrStaleSearchResult) {
				t.Fatalf("SaveExchange() error = %v, want %v", err, ErrStaleSearchResult)
			}
			if transactions.committed {
				t.Fatal("stale exchange transaction was committed")
			}
		})
	}
}

func TestSaveExchangeItemLockErrorRollsBack(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("lock items failed")
	queries := &fakeExchangeWriteQueries{
		exchangeID:   pgUUID(uuid.New()),
		lockItemsErr: databaseError,
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
		t.Fatal("failed item lock transaction was committed")
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
	rows                []db.FindExchangeNeighborsRow
	err                 error
	receivedItemID      pgtype.UUID
	blockConflict       bool
	blockConflictErr    error
	blockConflictParams db.HasUserBlockConflictParams
	statsRows           []db.ListExchangeSearchUserStatsRow
	statsUserIDs        []pgtype.UUID
	statsErr            error
}

type fakeExchangeWriteQueries struct {
	exchangeID     pgtype.UUID
	createErr      error
	signature      string
	compositionKey string
	participantErr error
	participants   []db.CreateExchangeParticipantParams

	chainStatus                  db.ChainStatus
	chainSignature               string
	chainComposition             string
	lockExchangeErr              error
	participantStatus            db.ParticipantStatus
	participantCompletedAt       pgtype.Timestamptz
	lockParticipantErr           error
	decisionItemsLocked          bool
	lockDecisionItemsErr         error
	accepted                     bool
	acceptErr                    error
	declined                     bool
	declineErr                   error
	refusals                     []db.RecordItemRefusalParams
	pending                      int64
	pendingErr                   error
	items                        []db.LockExchangeItemsRow
	lockItemsErr                 error
	reserved                     int64
	reserveErr                   error
	confirmed                    bool
	confirmErr                   error
	cancelled                    bool
	cancelErr                    error
	released                     int64
	releaseErr                   error
	releaseCalled                bool
	dealsBrokenUpdated           int64
	dealsBrokenErr               error
	dealsBrokenCalled            bool
	competingCancelled           int64
	cancelCompetingErr           error
	cancelCompetingCalled        bool
	completionParticipant        db.LockExchangeCompletionParticipantRow
	lockCompletionParticipantErr error
	completionConfirmed          bool
	confirmCompletionErr         error
	incomplete                   int64
	incompleteErr                error
	traded                       int64
	tradedErr                    error
	completed                    bool
	completeErr                  error
	dealsCompletedUpdated        int64
	dealsCompletedErr            error
	systemMessages               []db.CreateChainSystemMessageParams
	systemMessageErr             error
	pickupLocked                 bool
	lockPickupErr                error
	confirmedExchangeID          pgtype.UUID
	findConfirmedErr             error
	pickupSet                    []db.SetItemPickupPointParams
	pickupUpdated                int64
	setPickupErr                 error
	promoted                     int64
	promoteErr                   error
	delivered                    int64
	deliverErr                   error
	deliverCalled                bool
	pickupCleared                bool
	clearPickupErr               error
	adminAudit                   []db.CreateAdminAuditLogParams
	adminAuditErr                error
}

func (f *fakeExchangeWriteQueries) CreateAdminAuditLog(
	_ context.Context,
	params db.CreateAdminAuditLogParams,
) (db.AdminAuditLog, error) {
	f.adminAudit = append(f.adminAudit, params)
	return db.AdminAuditLog{}, f.adminAuditErr
}

func (f *fakeExchangeWriteQueries) LockItemPickup(context.Context, pgtype.UUID) error {
	f.pickupLocked = true
	return f.lockPickupErr
}

// Невыставленный confirmedExchangeID означает «вещь ни в каком подтверждённом обмене»,
// и база в этом случае отвечает именно ErrNoRows.
func (f *fakeExchangeWriteQueries) FindConfirmedExchangeForItem(
	context.Context,
	pgtype.UUID,
) (pgtype.UUID, error) {
	if f.findConfirmedErr != nil {
		return pgtype.UUID{}, f.findConfirmedErr
	}
	if !f.confirmedExchangeID.Valid {
		return pgtype.UUID{}, pgx.ErrNoRows
	}

	return f.confirmedExchangeID, nil
}

func (f *fakeExchangeWriteQueries) SetItemPickupPoint(
	_ context.Context,
	params db.SetItemPickupPointParams,
) (int64, error) {
	f.pickupSet = append(f.pickupSet, params)
	return f.pickupUpdated, f.setPickupErr
}

func (f *fakeExchangeWriteQueries) PromoteExchangeToDelivering(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	return f.promoted, f.promoteErr
}

func (f *fakeExchangeWriteQueries) ClearExchangeItemsPickupPoint(
	context.Context,
	pgtype.UUID,
) error {
	f.pickupCleared = true
	return f.clearPickupErr
}

func (f *fakeExchangeWriteQueries) CreateChainSystemMessage(
	_ context.Context,
	params db.CreateChainSystemMessageParams,
) error {
	f.systemMessages = append(f.systemMessages, params)
	return f.systemMessageErr
}

// assertRecordedEvents проверяет и состав, и порядок событий: тред обмена читается как
// лента, поэтому «подтвердил» обязано стоять раньше «обмен подтверждён».
func assertRecordedEvents(
	t *testing.T,
	queries *fakeExchangeWriteQueries,
	want ...db.ChainMessageKind,
) {
	t.Helper()

	// Объявление через var, а не make: без событий срез остаётся nil и сходится с пустым
	// списком ожиданий, который приходит из вызова без аргументов.
	var recorded []db.ChainMessageKind
	for _, message := range queries.systemMessages {
		recorded = append(recorded, message.Kind)
	}

	if !reflect.DeepEqual(recorded, want) {
		t.Fatalf("recorded events = %v, want %v", recorded, want)
	}
}

func (f *fakeExchangeWriteQueries) CreateExchange(
	_ context.Context,
	params db.CreateExchangeParams,
) (pgtype.UUID, error) {
	f.signature = params.Signature
	f.compositionKey = params.CompositionKey
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
) (db.LockExchangeRow, error) {
	return db.LockExchangeRow{
		Status:         f.chainStatus,
		Signature:      f.chainSignature,
		CompositionKey: f.chainComposition,
	}, f.lockExchangeErr
}

func (f *fakeExchangeWriteQueries) LockExchangeDecisionItems(
	context.Context,
	pgtype.UUID,
) error {
	f.decisionItemsLocked = true
	return f.lockDecisionItemsErr
}

func (f *fakeExchangeWriteQueries) LockExchangeParticipant(
	context.Context,
	db.LockExchangeParticipantParams,
) (db.LockExchangeParticipantRow, error) {
	return db.LockExchangeParticipantRow{
		Status:                f.participantStatus,
		CompletionConfirmedAt: f.participantCompletedAt,
	}, f.lockParticipantErr
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

func (f *fakeExchangeWriteQueries) RecordItemRefusal(
	_ context.Context,
	params db.RecordItemRefusalParams,
) error {
	f.refusals = append(f.refusals, params)
	return nil
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

func (f *fakeExchangeWriteQueries) MarkExchangeDelivered(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	f.deliverCalled = true
	return f.delivered, f.deliverErr
}

func (f *fakeExchangeWriteQueries) CancelCompetingProposedExchanges(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	f.cancelCompetingCalled = true
	return f.competingCancelled, f.cancelCompetingErr
}

func (f *fakeExchangeWriteQueries) CancelExchange(context.Context, pgtype.UUID) error {
	f.cancelled = true
	return f.cancelErr
}

func (f *fakeExchangeWriteQueries) ReleaseExchangeItems(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	f.releaseCalled = true
	return f.released, f.releaseErr
}

func (f *fakeExchangeWriteQueries) IncrementUserDealsBroken(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	f.dealsBrokenCalled = true
	return f.dealsBrokenUpdated, f.dealsBrokenErr
}

func (f *fakeExchangeWriteQueries) LockExchangeCompletionParticipant(
	context.Context,
	db.LockExchangeCompletionParticipantParams,
) (db.LockExchangeCompletionParticipantRow, error) {
	return f.completionParticipant, f.lockCompletionParticipantErr
}

func (f *fakeExchangeWriteQueries) ConfirmExchangeParticipantCompletion(
	context.Context,
	db.ConfirmExchangeParticipantCompletionParams,
) error {
	f.completionConfirmed = true
	return f.confirmCompletionErr
}

func (f *fakeExchangeWriteQueries) CountIncompleteExchangeParticipants(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	return f.incomplete, f.incompleteErr
}

func (f *fakeExchangeWriteQueries) MarkExchangeItemsTraded(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	return f.traded, f.tradedErr
}

func (f *fakeExchangeWriteQueries) CompleteExchange(context.Context, pgtype.UUID) error {
	f.completed = true
	return f.completeErr
}

func (f *fakeExchangeWriteQueries) IncrementExchangeParticipantsDealsCompleted(
	context.Context,
	pgtype.UUID,
) (int64, error) {
	return f.dealsCompletedUpdated, f.dealsCompletedErr
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

func (f *fakeNeighborQueries) HasUserBlockConflict(
	_ context.Context,
	params db.HasUserBlockConflictParams,
) (bool, error) {
	f.blockConflictParams = params
	return f.blockConflict, f.blockConflictErr
}

func (f *fakeNeighborQueries) ListExchangeSearchUserStats(
	_ context.Context,
	userIDs []pgtype.UUID,
) ([]db.ListExchangeSearchUserStatsRow, error) {
	f.statsUserIDs = userIDs
	return f.statsRows, f.statsErr
}
