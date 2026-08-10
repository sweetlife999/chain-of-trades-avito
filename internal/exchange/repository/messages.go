package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

type messageQueries interface {
	GetExchangeAccess(context.Context, db.GetExchangeAccessParams) (db.GetExchangeAccessRow, error)
	CreateChainMessage(context.Context, db.CreateChainMessageParams) (db.CreateChainMessageRow, error)
	ListChainMessages(context.Context, pgtype.UUID) ([]db.ListChainMessagesRow, error)
	MarkChainMessagesRead(context.Context, db.MarkChainMessagesReadParams) error
}

// Статусы обмена, которые проверяет сервис. Через string(db.ChainStatus*), чтобы
// переименование значения в enum ловил компилятор, а не тест на живой БД. Типы из
// internal/db наружу при этом не протекают.
const (
	StatusProposed   = string(db.ChainStatusProposed)
	StatusConfirmed  = string(db.ChainStatusConfirmed)
	StatusDelivering = string(db.ChainStatusDelivering)
	StatusDelivered  = string(db.ChainStatusDelivered)
)

type messageRecord struct {
	ID             pgtype.UUID
	Kind           db.ChainMessageKind
	Body           pgtype.Text
	CreatedAt      pgtype.Timestamptz
	AuthorID       pgtype.UUID
	AuthorNickname pgtype.Text
	AuthorPhotoURL pgtype.Text
}

// ExchangeAccess отвечает, существует ли обмен, участвует ли в нём пользователь и в
// каком обмен состоянии. Три ответа нужны вместе: они раскладываются в 404, 403 и 409.
func (r *Repository) ExchangeAccess(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) (status string, isParticipant bool, err error) {
	access, err := r.messages.GetExchangeAccess(ctx, db.GetExchangeAccessParams{
		UserID:     pgUUID(userID),
		ExchangeID: pgUUID(exchangeID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, ErrNotFound
	}
	if err != nil {
		return "", false, fmt.Errorf("get exchange access: %w", err)
	}

	return string(access.Status), access.IsParticipant, nil
}

func (r *Repository) CreateMessage(
	ctx context.Context,
	exchangeID uuid.UUID,
	authorID uuid.UUID,
	body string,
) (exchangemodel.Message, error) {
	row, err := r.messages.CreateChainMessage(ctx, db.CreateChainMessageParams{
		ExchangeID: pgUUID(exchangeID),
		AuthorID:   pgUUID(authorID),
		Body:       pgtype.Text{String: body, Valid: true},
	})
	if err != nil {
		return exchangemodel.Message{}, fmt.Errorf("create chain message: %w", err)
	}

	return messageFromRecord(recordFromCreateRow(row)), nil
}

func (r *Repository) ListMessages(
	ctx context.Context,
	exchangeID uuid.UUID,
) ([]exchangemodel.Message, error) {
	rows, err := r.messages.ListChainMessages(ctx, pgUUID(exchangeID))
	if err != nil {
		return nil, fmt.Errorf("list chain messages: %w", err)
	}

	messages := make([]exchangemodel.Message, len(rows))
	for index, row := range rows {
		messages[index] = messageFromRecord(recordFromListMessageRow(row))
	}

	return messages, nil
}

// MarkMessagesRead двигает отметку участника до lastMessageID. Сообщение из чужого треда
// или уже пройденное запрос игнорирует, поэтому проверять их отдельно не нужно.
func (r *Repository) MarkMessagesRead(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
	lastMessageID uuid.UUID,
) error {
	err := r.messages.MarkChainMessagesRead(ctx, db.MarkChainMessagesReadParams{
		ExchangeID:    pgUUID(exchangeID),
		UserID:        pgUUID(userID),
		LastMessageID: pgUUID(lastMessageID),
	})
	if err != nil {
		return fmt.Errorf("mark chain messages read: %w", err)
	}

	return nil
}

// recordExchangeEvent пишет событие сделки в тред обмена. Вызывается внутри транзакции,
// которая это событие и породила: тред не может разойтись со статусами участников.
// Пустой author — событие принадлежит всему обмену, а не конкретному человеку.
func recordExchangeEvent(
	ctx context.Context,
	queries exchangeWriteQueries,
	exchangeID uuid.UUID,
	author pgtype.UUID,
	kind db.ChainMessageKind,
) error {
	err := queries.CreateChainSystemMessage(ctx, db.CreateChainSystemMessageParams{
		ExchangeID: pgUUID(exchangeID),
		AuthorID:   author,
		Kind:       kind,
	})
	if err != nil {
		return fmt.Errorf("record %s event: %w", kind, err)
	}

	return nil
}

func messageFromRecord(record messageRecord) exchangemodel.Message {
	message := exchangemodel.Message{
		ID:        uuid.UUID(record.ID.Bytes),
		Kind:      string(record.Kind),
		Body:      optionalText(record.Body),
		CreatedAt: record.CreatedAt.Time,
	}

	if record.AuthorID.Valid {
		message.Author = &exchangemodel.ParticipantUser{
			ID:       uuid.UUID(record.AuthorID.Bytes),
			Nickname: record.AuthorNickname.String,
			PhotoURL: optionalText(record.AuthorPhotoURL),
		}
	}

	return message
}

func recordFromCreateRow(row db.CreateChainMessageRow) messageRecord {
	return messageRecord{
		ID:             row.ID,
		Kind:           row.Kind,
		Body:           row.Body,
		CreatedAt:      row.CreatedAt,
		AuthorID:       row.AuthorID,
		AuthorNickname: row.AuthorNickname,
		AuthorPhotoURL: row.AuthorPhotoUrl,
	}
}

func recordFromListMessageRow(row db.ListChainMessagesRow) messageRecord {
	return messageRecord{
		ID:             row.ID,
		Kind:           row.Kind,
		Body:           row.Body,
		CreatedAt:      row.CreatedAt,
		AuthorID:       row.AuthorID,
		AuthorNickname: row.AuthorNickname,
		AuthorPhotoURL: row.AuthorPhotoUrl,
	}
}
