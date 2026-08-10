package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

func TestListByUserGroupsParticipants(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	firstExchangeID := uuid.New()
	secondExchangeID := uuid.New()
	firstParticipant := listExchangeRow(firstExchangeID, uuid.New(), 0)
	firstParticipant.UnreadCount = 3
	secondParticipant := listExchangeRow(firstExchangeID, uuid.New(), 1)
	secondParticipant.UnreadCount = 3
	thirdParticipant := listExchangeRow(secondExchangeID, userID, 0)

	queries := &fakeExchangeReadQueries{listRows: []db.ListExchangesByUserRow{
		firstParticipant,
		secondParticipant,
		thirdParticipant,
	}}
	repository := newRepositoryWithReads(queries)

	exchanges, err := repository.ListByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}

	if queries.listUserID != pgUUID(userID) {
		t.Fatalf("ListExchangesByUser() user ID = %v, want %v", queries.listUserID, pgUUID(userID))
	}

	if len(exchanges) != 2 {
		t.Fatalf("exchange count = %d, want 2", len(exchanges))
	}

	if exchanges[0].ID != firstExchangeID || len(exchanges[0].Participants) != 2 {
		t.Fatalf("first exchange = %+v, want ID %s with 2 participants", exchanges[0], firstExchangeID)
	}

	// Счётчик приходит одинаковым в строке каждого участника и относится к обмену целиком.
	if exchanges[0].UnreadCount != 3 {
		t.Fatalf("unread count = %d, want 3", exchanges[0].UnreadCount)
	}

	if exchanges[1].ID != secondExchangeID || len(exchanges[1].Participants) != 1 {
		t.Fatalf("second exchange = %+v, want ID %s with 1 participant", exchanges[1], secondExchangeID)
	}

	participant := exchanges[0].Participants[0]
	if participant.User.ID != uuid.UUID(firstParticipant.UserID.Bytes) ||
		participant.GivesItem.ID != uuid.UUID(firstParticipant.GivesItemID.Bytes) ||
		participant.ReceivesItem.ID != uuid.UUID(firstParticipant.ReceivesItemID.Bytes) {
		t.Fatalf("mapped participant = %+v", participant)
	}

	if participant.GivesItem.Category.Slug != "books" ||
		participant.ReceivesItem.Category.Slug != "hobby" {
		t.Fatalf("mapped categories = %+v / %+v", participant.GivesItem.Category, participant.ReceivesItem.Category)
	}
}

func TestListByUserReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	repository := newRepositoryWithReads(&fakeExchangeReadQueries{})
	exchanges, err := repository.ListByUser(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}

	if exchanges == nil {
		t.Fatal("ListByUser() returned nil, want []")
	}
}

func TestListByUserError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := newRepositoryWithReads(&fakeExchangeReadQueries{listErr: databaseError})

	_, err := repository.ListByUser(context.Background(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("ListByUser() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestListActiveByUserForAdminUsesPaginationAndGroupsParticipants(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	exchangeID := uuid.New()
	first := activeExchangeRow(exchangeID, userID, 0)
	second := activeExchangeRow(exchangeID, uuid.New(), 1)
	queries := &fakeExchangeReadQueries{
		activeRows:  []db.ListActiveExchangesForAdminRow{first, second},
		activeCount: 4,
	}
	repository := newRepositoryWithReads(queries)

	exchanges, err := repository.ListActiveByUser(context.Background(), userID, 2, 1)
	if err != nil {
		t.Fatalf("ListActiveByUser() error = %v", err)
	}
	if queries.activeParams != (db.ListActiveExchangesForAdminParams{
		UserID: pgUUID(userID), ExchangeStatus: "", PageLimit: 2, PageOffset: 1,
	}) {
		t.Fatalf("params = %+v", queries.activeParams)
	}
	if len(exchanges) != 1 || exchanges[0].ID != exchangeID || len(exchanges[0].Participants) != 2 {
		t.Fatalf("exchanges = %+v", exchanges)
	}

	total, err := repository.CountActiveByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("CountActiveByUser() error = %v", err)
	}
	if total != 4 || queries.countParams != (db.CountActiveExchangesForAdminParams{
		UserID: pgUUID(userID), ExchangeStatus: "",
	}) {
		t.Fatalf("total = %d, count params = %+v", total, queries.countParams)
	}
}

func TestListActiveForAdminFiltersByStatus(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	queries := &fakeExchangeReadQueries{
		activeRows:  []db.ListActiveExchangesForAdminRow{activeExchangeRow(exchangeID, uuid.New(), 0)},
		activeCount: 1,
	}
	repository := newRepositoryWithReads(queries)

	exchanges, err := repository.ListActiveForAdmin(context.Background(), "delivering", 10, 5)
	if err != nil {
		t.Fatalf("ListActiveForAdmin() error = %v", err)
	}
	if queries.activeParams != (db.ListActiveExchangesForAdminParams{
		ExchangeStatus: "delivering", PageLimit: 10, PageOffset: 5,
	}) {
		t.Fatalf("params = %+v", queries.activeParams)
	}
	if len(exchanges) != 1 || exchanges[0].ID != exchangeID {
		t.Fatalf("exchanges = %+v", exchanges)
	}

	total, err := repository.CountActiveForAdmin(context.Background(), "delivering")
	if err != nil {
		t.Fatalf("CountActiveForAdmin() error = %v", err)
	}
	if total != 1 || queries.countParams != (db.CountActiveExchangesForAdminParams{
		ExchangeStatus: "delivering",
	}) {
		t.Fatalf("total = %d, count params = %+v", total, queries.countParams)
	}
}

func TestGetByID(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	rows := []db.GetExchangeByIDRow{
		getExchangeRow(exchangeID, uuid.New(), 0),
		getExchangeRow(exchangeID, uuid.New(), 1),
	}
	rows[0].ExchangeStatus = db.ChainStatusCancelled
	rows[1].ExchangeStatus = db.ChainStatusCancelled
	rows[0].ExchangeCancelReason = db.NullChainCancelReason{
		ChainCancelReason: db.ChainCancelReasonConfirmedBroken,
		Valid:             true,
	}
	rows[1].ExchangeCancelReason = rows[0].ExchangeCancelReason
	queries := &fakeExchangeReadQueries{getRows: rows}
	repository := newRepositoryWithReads(queries)

	exchange, err := repository.GetByID(context.Background(), exchangeID, uuid.New())
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if queries.getExchangeID != pgUUID(exchangeID) {
		t.Fatalf("GetExchangeByID() ID = %v, want %v", queries.getExchangeID, pgUUID(exchangeID))
	}

	if exchange.ID != exchangeID || len(exchange.Participants) != 2 {
		t.Fatalf("GetByID() = %+v, want ID %s with 2 participants", exchange, exchangeID)
	}
	if exchange.CancelReason == nil || *exchange.CancelReason != string(db.ChainCancelReasonConfirmedBroken) {
		t.Fatalf("CancelReason = %v, want %q", exchange.CancelReason, db.ChainCancelReasonConfirmedBroken)
	}

	closedAt := rows[0].ExchangeClosedAt.Time
	if exchange.ClosedAt == nil || !exchange.ClosedAt.Equal(closedAt) {
		t.Fatalf("ClosedAt = %v, want %v", exchange.ClosedAt, closedAt)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	t.Parallel()

	repository := newRepositoryWithReads(&fakeExchangeReadQueries{})
	_, err := repository.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, ErrNotFound)
	}
}

func TestGetByIDError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := newRepositoryWithReads(&fakeExchangeReadQueries{getErr: databaseError})

	_, err := repository.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, databaseError) {
		t.Fatalf("GetByID() error = %v, want wrapped %v", err, databaseError)
	}
}

func TestOptionalValues(t *testing.T) {
	t.Parallel()

	if optionalTime(pgtype.Timestamptz{}) != nil {
		t.Fatal("optionalTime() for NULL must return nil")
	}
	if optionalText(pgtype.Text{}) != nil {
		t.Fatal("optionalText() for NULL must return nil")
	}
	if optionalCancelReason(db.NullChainCancelReason{}) != nil {
		t.Fatal("optionalCancelReason() for NULL must return nil")
	}

	now := time.Now().UTC()
	text := "photo"
	if !reflect.DeepEqual(optionalTime(pgtype.Timestamptz{Time: now, Valid: true}), &now) {
		t.Fatal("optionalTime() lost a valid value")
	}
	if !reflect.DeepEqual(optionalText(pgtype.Text{String: text, Valid: true}), &text) {
		t.Fatal("optionalText() lost a valid value")
	}
	wantReason := string(db.ChainCancelReasonProposalDeclined)
	if !reflect.DeepEqual(optionalCancelReason(db.NullChainCancelReason{
		ChainCancelReason: db.ChainCancelReasonProposalDeclined,
		Valid:             true,
	}), &wantReason) {
		t.Fatal("optionalCancelReason() lost a valid value")
	}
}

type fakeExchangeReadQueries struct {
	listRows      []db.ListExchangesByUserRow
	listErr       error
	listUserID    pgtype.UUID
	activeRows    []db.ListActiveExchangesForAdminRow
	activeErr     error
	activeParams  db.ListActiveExchangesForAdminParams
	activeCount   int64
	countErr      error
	countParams   db.CountActiveExchangesForAdminParams
	getRows       []db.GetExchangeByIDRow
	getErr        error
	getExchangeID pgtype.UUID
	getUserID     pgtype.UUID
}

func (f *fakeExchangeReadQueries) ListActiveExchangesForAdmin(
	_ context.Context,
	params db.ListActiveExchangesForAdminParams,
) ([]db.ListActiveExchangesForAdminRow, error) {
	f.activeParams = params
	return f.activeRows, f.activeErr
}

func (f *fakeExchangeReadQueries) CountActiveExchangesForAdmin(
	_ context.Context,
	params db.CountActiveExchangesForAdminParams,
) (int64, error) {
	f.countParams = params
	return f.activeCount, f.countErr
}

func (f *fakeExchangeReadQueries) ListExchangesByUser(
	_ context.Context,
	userID pgtype.UUID,
) ([]db.ListExchangesByUserRow, error) {
	f.listUserID = userID
	return f.listRows, f.listErr
}

func (f *fakeExchangeReadQueries) GetExchangeByID(
	_ context.Context,
	params db.GetExchangeByIDParams,
) ([]db.GetExchangeByIDRow, error) {
	f.getExchangeID = params.ExchangeID
	f.getUserID = params.UserID
	return f.getRows, f.getErr
}

func listExchangeRow(exchangeID, userID uuid.UUID, position int32) db.ListExchangesByUserRow {
	createdAt := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	return db.ListExchangesByUserRow{
		ExchangeID:              pgUUID(exchangeID),
		ExchangeStatus:          db.ChainStatusProposed,
		ExchangeCreatedAt:       pgtype.Timestamptz{Time: createdAt, Valid: true},
		ExchangeUpdatedAt:       pgtype.Timestamptz{Time: createdAt, Valid: true},
		UserID:                  pgUUID(userID),
		Position:                position,
		ParticipantStatus:       db.ParticipantStatusPending,
		Nickname:                "user",
		GivesItemID:             pgUUID(uuid.New()),
		GivesItemTitle:          "Book",
		GivesItemDescription:    "Description",
		GivesItemStatus:         db.ItemStatusAvailable,
		GivesCategorySlug:       "books",
		GivesCategoryName:       "Books",
		ReceivesItemID:          pgUUID(uuid.New()),
		ReceivesItemTitle:       "Game",
		ReceivesItemDescription: "Description",
		ReceivesItemStatus:      db.ItemStatusAvailable,
		ReceivesCategorySlug:    "hobby",
		ReceivesCategoryName:    "Hobby",
	}
}

func getExchangeRow(exchangeID, userID uuid.UUID, position int32) db.GetExchangeByIDRow {
	listRow := listExchangeRow(exchangeID, userID, position)
	closedAt := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	return db.GetExchangeByIDRow{
		ExchangeID:              listRow.ExchangeID,
		ExchangeStatus:          listRow.ExchangeStatus,
		ExchangeCreatedAt:       listRow.ExchangeCreatedAt,
		ExchangeUpdatedAt:       listRow.ExchangeUpdatedAt,
		ExchangeClosedAt:        pgtype.Timestamptz{Time: closedAt, Valid: true},
		UserID:                  listRow.UserID,
		Position:                listRow.Position,
		ParticipantStatus:       listRow.ParticipantStatus,
		Nickname:                listRow.Nickname,
		GivesItemID:             listRow.GivesItemID,
		GivesItemTitle:          listRow.GivesItemTitle,
		GivesItemDescription:    listRow.GivesItemDescription,
		GivesItemStatus:         listRow.GivesItemStatus,
		GivesCategorySlug:       listRow.GivesCategorySlug,
		GivesCategoryName:       listRow.GivesCategoryName,
		ReceivesItemID:          listRow.ReceivesItemID,
		ReceivesItemTitle:       listRow.ReceivesItemTitle,
		ReceivesItemDescription: listRow.ReceivesItemDescription,
		ReceivesItemStatus:      listRow.ReceivesItemStatus,
		ReceivesCategorySlug:    listRow.ReceivesCategorySlug,
		ReceivesCategoryName:    listRow.ReceivesCategoryName,
	}
}

func activeExchangeRow(
	exchangeID uuid.UUID,
	userID uuid.UUID,
	position int32,
) db.ListActiveExchangesForAdminRow {
	row := listExchangeRow(exchangeID, userID, position)
	return db.ListActiveExchangesForAdminRow{
		ExchangeID:              row.ExchangeID,
		ExchangeStatus:          row.ExchangeStatus,
		ExchangeCreatedAt:       row.ExchangeCreatedAt,
		ExchangeUpdatedAt:       row.ExchangeUpdatedAt,
		ExchangeClosedAt:        row.ExchangeClosedAt,
		UserID:                  row.UserID,
		Position:                row.Position,
		ParticipantStatus:       row.ParticipantStatus,
		DecidedAt:               row.DecidedAt,
		CompletionConfirmedAt:   row.CompletionConfirmedAt,
		Nickname:                row.Nickname,
		UserPhotoUrl:            row.UserPhotoUrl,
		GivesItemID:             row.GivesItemID,
		GivesItemTitle:          row.GivesItemTitle,
		GivesItemDescription:    row.GivesItemDescription,
		GivesItemStatus:         row.GivesItemStatus,
		GivesCategorySlug:       row.GivesCategorySlug,
		GivesCategoryName:       row.GivesCategoryName,
		ReceivesItemID:          row.ReceivesItemID,
		ReceivesItemTitle:       row.ReceivesItemTitle,
		ReceivesItemDescription: row.ReceivesItemDescription,
		ReceivesItemStatus:      row.ReceivesItemStatus,
		ReceivesCategorySlug:    row.ReceivesCategorySlug,
		ReceivesCategoryName:    row.ReceivesCategoryName,
	}
}
