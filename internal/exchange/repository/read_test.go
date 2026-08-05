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
	secondParticipant := listExchangeRow(firstExchangeID, uuid.New(), 1)
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

func TestGetByID(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	rows := []db.GetExchangeByIDRow{
		getExchangeRow(exchangeID, uuid.New(), 0),
		getExchangeRow(exchangeID, uuid.New(), 1),
	}
	queries := &fakeExchangeReadQueries{getRows: rows}
	repository := newRepositoryWithReads(queries)

	exchange, err := repository.GetByID(context.Background(), exchangeID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if queries.getExchangeID != pgUUID(exchangeID) {
		t.Fatalf("GetExchangeByID() ID = %v, want %v", queries.getExchangeID, pgUUID(exchangeID))
	}

	if exchange.ID != exchangeID || len(exchange.Participants) != 2 {
		t.Fatalf("GetByID() = %+v, want ID %s with 2 participants", exchange, exchangeID)
	}

	closedAt := rows[0].ExchangeClosedAt.Time
	if exchange.ClosedAt == nil || !exchange.ClosedAt.Equal(closedAt) {
		t.Fatalf("ClosedAt = %v, want %v", exchange.ClosedAt, closedAt)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	t.Parallel()

	repository := newRepositoryWithReads(&fakeExchangeReadQueries{})
	_, err := repository.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, ErrNotFound)
	}
}

func TestGetByIDError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	repository := newRepositoryWithReads(&fakeExchangeReadQueries{getErr: databaseError})

	_, err := repository.GetByID(context.Background(), uuid.New())
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

	now := time.Now().UTC()
	text := "photo"
	if !reflect.DeepEqual(optionalTime(pgtype.Timestamptz{Time: now, Valid: true}), &now) {
		t.Fatal("optionalTime() lost a valid value")
	}
	if !reflect.DeepEqual(optionalText(pgtype.Text{String: text, Valid: true}), &text) {
		t.Fatal("optionalText() lost a valid value")
	}
}

type fakeExchangeReadQueries struct {
	listRows      []db.ListExchangesByUserRow
	listErr       error
	listUserID    pgtype.UUID
	getRows       []db.GetExchangeByIDRow
	getErr        error
	getExchangeID pgtype.UUID
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
	exchangeID pgtype.UUID,
) ([]db.GetExchangeByIDRow, error) {
	f.getExchangeID = exchangeID
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
