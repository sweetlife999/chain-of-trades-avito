package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

func TestExchangeAccessReturnsStatusAndParticipation(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	userID := uuid.New()
	queries := &fakeMessageQueries{
		access: db.GetExchangeAccessRow{Status: db.ChainStatusConfirmed, IsParticipant: true},
	}

	status, isParticipant, err := newRepositoryWithMessages(queries).
		ExchangeAccess(context.Background(), exchangeID, userID)
	if err != nil {
		t.Fatalf("ExchangeAccess() error = %v", err)
	}

	if status != "confirmed" || !isParticipant {
		t.Fatalf("ExchangeAccess() = (%q, %t), want (confirmed, true)", status, isParticipant)
	}
	if queries.accessParams.ExchangeID != pgUUID(exchangeID) ||
		queries.accessParams.UserID != pgUUID(userID) {
		t.Fatalf("GetExchangeAccess() params = %+v", queries.accessParams)
	}
}

func TestExchangeAccessMissingExchange(t *testing.T) {
	t.Parallel()

	queries := &fakeMessageQueries{accessErr: pgx.ErrNoRows}

	_, _, err := newRepositoryWithMessages(queries).
		ExchangeAccess(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ExchangeAccess() error = %v, want ErrNotFound", err)
	}
}

func TestCreateMessageMapsAuthor(t *testing.T) {
	t.Parallel()

	messageID := uuid.New()
	authorID := uuid.New()
	createdAt := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	photo := "https://example.com/avatar.jpg"
	queries := &fakeMessageQueries{created: db.CreateChainMessageRow{
		ID:             pgUUID(messageID),
		Kind:           db.ChainMessageKindText,
		Body:           pgtype.Text{String: "заберу в субботу", Valid: true},
		CreatedAt:      pgtype.Timestamptz{Time: createdAt, Valid: true},
		AuthorID:       pgUUID(authorID),
		AuthorNickname: pgtype.Text{String: "samir", Valid: true},
		AuthorPhotoUrl: pgtype.Text{String: photo, Valid: true},
	}}

	message, err := newRepositoryWithMessages(queries).
		CreateMessage(context.Background(), uuid.New(), authorID, "заберу в субботу")
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}

	if message.ID != messageID || message.Kind != "text" || message.CreatedAt != createdAt {
		t.Fatalf("CreateMessage() = %+v", message)
	}
	if message.Body == nil || *message.Body != "заберу в субботу" {
		t.Fatalf("CreateMessage() body = %v", message.Body)
	}
	if message.Author == nil || message.Author.ID != authorID || message.Author.Nickname != "samir" {
		t.Fatalf("CreateMessage() author = %+v", message.Author)
	}
	if message.Author.PhotoURL == nil || *message.Author.PhotoURL != photo {
		t.Fatalf("CreateMessage() author photo = %v", message.Author.PhotoURL)
	}
	if queries.createParams.Body != (pgtype.Text{String: "заберу в субботу", Valid: true}) {
		t.Fatalf("CreateChainMessage() body param = %+v", queries.createParams.Body)
	}
}

func TestListMessagesLeavesSystemEventsWithoutAuthor(t *testing.T) {
	t.Parallel()

	exchangeID := uuid.New()
	authorID := uuid.New()
	queries := &fakeMessageQueries{list: []db.ListChainMessagesRow{
		{
			ID:             pgUUID(uuid.New()),
			Kind:           db.ChainMessageKindText,
			Body:           pgtype.Text{String: "привет", Valid: true},
			CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
			AuthorID:       pgUUID(authorID),
			AuthorNickname: pgtype.Text{String: "samir", Valid: true},
		},
		{
			ID:        pgUUID(uuid.New()),
			Kind:      db.ChainMessageKindExchangeConfirmed,
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}}

	messages, err := newRepositoryWithMessages(queries).
		ListMessages(context.Background(), exchangeID)
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(messages))
	}
	if messages[0].Author == nil || messages[0].Author.ID != authorID {
		t.Fatalf("first message author = %+v, want the participant", messages[0].Author)
	}
	if messages[1].Author != nil {
		t.Fatalf("exchange event author = %+v, want none", messages[1].Author)
	}
	if messages[1].Body != nil {
		t.Fatalf("exchange event body = %v, want none", messages[1].Body)
	}
	if messages[1].Kind != "exchange_confirmed" {
		t.Fatalf("exchange event kind = %q", messages[1].Kind)
	}
	if queries.listExchangeID != pgUUID(exchangeID) {
		t.Fatalf("ListChainMessages() exchange ID = %v, want %v", queries.listExchangeID, pgUUID(exchangeID))
	}
}

func TestListMessagesReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	messages, err := newRepositoryWithMessages(&fakeMessageQueries{}).
		ListMessages(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}

	if messages == nil {
		t.Fatal("ListMessages() = nil, want an empty slice so the API answers []")
	}
}

type fakeMessageQueries struct {
	access         db.GetExchangeAccessRow
	accessErr      error
	accessParams   db.GetExchangeAccessParams
	created        db.CreateChainMessageRow
	createErr      error
	createParams   db.CreateChainMessageParams
	list           []db.ListChainMessagesRow
	listErr        error
	listExchangeID pgtype.UUID
}

func (f *fakeMessageQueries) GetExchangeAccess(
	_ context.Context,
	params db.GetExchangeAccessParams,
) (db.GetExchangeAccessRow, error) {
	f.accessParams = params
	return f.access, f.accessErr
}

func (f *fakeMessageQueries) CreateChainMessage(
	_ context.Context,
	params db.CreateChainMessageParams,
) (db.CreateChainMessageRow, error) {
	f.createParams = params
	return f.created, f.createErr
}

func (f *fakeMessageQueries) ListChainMessages(
	_ context.Context,
	exchangeID pgtype.UUID,
) ([]db.ListChainMessagesRow, error) {
	f.listExchangeID = exchangeID
	return f.list, f.listErr
}
