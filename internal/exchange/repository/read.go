package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

var ErrNotFound = errors.New("exchange not found")

type exchangeReadQueries interface {
	ListExchangesByUser(context.Context, pgtype.UUID) ([]db.ListExchangesByUserRow, error)
	ListActiveExchangesByUserForAdmin(
		context.Context,
		db.ListActiveExchangesByUserForAdminParams,
	) ([]db.ListActiveExchangesByUserForAdminRow, error)
	CountActiveExchangesByUserForAdmin(context.Context, pgtype.UUID) (int64, error)
	GetExchangeByID(context.Context, db.GetExchangeByIDParams) ([]db.GetExchangeByIDRow, error)
}

type exchangeRecord struct {
	ExchangeID              pgtype.UUID
	ExchangeStatus          db.ChainStatus
	ExchangeCreatedAt       pgtype.Timestamptz
	ExchangeUpdatedAt       pgtype.Timestamptz
	ExchangeClosedAt        pgtype.Timestamptz
	UserID                  pgtype.UUID
	Position                int32
	ParticipantStatus       db.ParticipantStatus
	DecidedAt               pgtype.Timestamptz
	CompletionConfirmedAt   pgtype.Timestamptz
	Nickname                string
	UserPhotoURL            pgtype.Text
	GivesItemID             pgtype.UUID
	GivesItemTitle          string
	GivesItemDescription    string
	GivesItemStatus         db.ItemStatus
	GivesCategorySlug       string
	GivesCategoryName       string
	ReceivesItemID          pgtype.UUID
	ReceivesItemTitle       string
	ReceivesItemDescription string
	ReceivesItemStatus      db.ItemStatus
	ReceivesCategorySlug    string
	ReceivesCategoryName    string
	UnreadCount             int64
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]exchangemodel.Details, error) {
	rows, err := r.reads.ListExchangesByUser(ctx, pgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list exchanges by user: %w", err)
	}

	records := make([]exchangeRecord, len(rows))
	for index, row := range rows {
		records[index] = recordFromListRow(row)
	}

	return groupExchangeRecords(records), nil
}

func (r *Repository) ListActiveByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit int32,
	offset int32,
) ([]exchangemodel.Details, error) {
	rows, err := r.reads.ListActiveExchangesByUserForAdmin(
		ctx,
		db.ListActiveExchangesByUserForAdminParams{
			UserID:     pgUUID(userID),
			PageLimit:  limit,
			PageOffset: offset,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list active exchanges by user: %w", err)
	}

	records := make([]exchangeRecord, len(rows))
	for index, row := range rows {
		records[index] = recordFromAdminListRow(row)
	}

	return groupExchangeRecords(records), nil
}

func (r *Repository) CountActiveByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	total, err := r.reads.CountActiveExchangesByUserForAdmin(ctx, pgUUID(userID))
	if err != nil {
		return 0, fmt.Errorf("count active exchanges by user: %w", err)
	}

	return total, nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	exchangeID uuid.UUID,
	userID uuid.UUID,
) (exchangemodel.Details, error) {
	rows, err := r.reads.GetExchangeByID(ctx, db.GetExchangeByIDParams{
		ExchangeID: pgUUID(exchangeID),
		UserID:     pgUUID(userID),
	})
	if err != nil {
		return exchangemodel.Details{}, fmt.Errorf("get exchange by ID: %w", err)
	}

	if len(rows) == 0 {
		return exchangemodel.Details{}, ErrNotFound
	}

	records := make([]exchangeRecord, len(rows))
	for index, row := range rows {
		records[index] = recordFromGetRow(row)
	}

	exchanges := groupExchangeRecords(records)
	return exchanges[0], nil
}

func groupExchangeRecords(records []exchangeRecord) []exchangemodel.Details {
	exchanges := make([]exchangemodel.Details, 0)
	indexes := make(map[uuid.UUID]int)

	for _, record := range records {
		exchangeID := uuid.UUID(record.ExchangeID.Bytes)
		index, exists := indexes[exchangeID]
		if !exists {
			index = len(exchanges)
			indexes[exchangeID] = index
			exchanges = append(exchanges, exchangemodel.Details{
				ID:           exchangeID,
				Status:       string(record.ExchangeStatus),
				Participants: make([]exchangemodel.DetailsParticipant, 0),
				UnreadCount:  record.UnreadCount,
				CreatedAt:    record.ExchangeCreatedAt.Time,
				UpdatedAt:    record.ExchangeUpdatedAt.Time,
				ClosedAt:     optionalTime(record.ExchangeClosedAt),
			})
		}

		exchanges[index].Participants = append(
			exchanges[index].Participants,
			participantFromRecord(record),
		)
	}

	return exchanges
}

func participantFromRecord(record exchangeRecord) exchangemodel.DetailsParticipant {
	return exchangemodel.DetailsParticipant{
		User: exchangemodel.ParticipantUser{
			ID:       uuid.UUID(record.UserID.Bytes),
			Nickname: record.Nickname,
			PhotoURL: optionalText(record.UserPhotoURL),
		},
		GivesItem: exchangemodel.ParticipantItem{
			ID:          uuid.UUID(record.GivesItemID.Bytes),
			Title:       record.GivesItemTitle,
			Description: record.GivesItemDescription,
			Status:      string(record.GivesItemStatus),
			Category: exchangemodel.ParticipantCategory{
				Slug: record.GivesCategorySlug,
				Name: record.GivesCategoryName,
			},
		},
		ReceivesItem: exchangemodel.ParticipantItem{
			ID:          uuid.UUID(record.ReceivesItemID.Bytes),
			Title:       record.ReceivesItemTitle,
			Description: record.ReceivesItemDescription,
			Status:      string(record.ReceivesItemStatus),
			Category: exchangemodel.ParticipantCategory{
				Slug: record.ReceivesCategorySlug,
				Name: record.ReceivesCategoryName,
			},
		},
		Position:              record.Position,
		Status:                string(record.ParticipantStatus),
		DecidedAt:             optionalTime(record.DecidedAt),
		CompletionConfirmedAt: optionalTime(record.CompletionConfirmedAt),
	}
}

func recordFromListRow(row db.ListExchangesByUserRow) exchangeRecord {
	return exchangeRecord{
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
		UserPhotoURL:            row.UserPhotoUrl,
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
		UnreadCount:             row.UnreadCount,
	}
}

func recordFromAdminListRow(row db.ListActiveExchangesByUserForAdminRow) exchangeRecord {
	return exchangeRecord{
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
		UserPhotoURL:            row.UserPhotoUrl,
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
		UnreadCount:             row.UnreadCount,
	}
}

func recordFromGetRow(row db.GetExchangeByIDRow) exchangeRecord {
	return exchangeRecord{
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
		UserPhotoURL:            row.UserPhotoUrl,
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
		UnreadCount:             row.UnreadCount,
	}
}

func optionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	result := value.Time
	return &result
}

func optionalText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	result := value.String
	return &result
}
