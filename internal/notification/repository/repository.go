package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	notificationmodel "github.com/sweetlife999/chain-of-trades-avito/internal/notification/model"
)

type Queries interface {
	ListNotifications(context.Context, db.ListNotificationsParams) ([]db.ListNotificationsRow, error)
	CountUnreadNotifications(context.Context, pgtype.UUID) (int64, error)
	MarkNotificationRead(context.Context, db.MarkNotificationReadParams) (int64, error)
	MarkAllNotificationsRead(context.Context, pgtype.UUID) (int64, error)
}

type Repository struct {
	queries Queries
}

func New(queries Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) List(
	ctx context.Context,
	userID uuid.UUID,
	filter notificationmodel.Filter,
) ([]notificationmodel.Notification, error) {
	rows, err := r.queries.ListNotifications(ctx, db.ListNotificationsParams{
		UserID:     pgUUID(userID),
		UnreadOnly: filter.UnreadOnly,
		PageOffset: filter.Offset,
		PageLimit:  filter.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}

	notifications := make([]notificationmodel.Notification, len(rows))
	for index, row := range rows {
		notifications[index] = fromRow(row)
	}

	return notifications, nil
}

func (r *Repository) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := r.queries.CountUnreadNotifications(ctx, pgUUID(userID))
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *Repository) MarkRead(
	ctx context.Context,
	userID uuid.UUID,
	notificationID uuid.UUID,
) (bool, error) {
	affected, err := r.queries.MarkNotificationRead(ctx, db.MarkNotificationReadParams{
		NotificationID: pgUUID(notificationID),
		UserID:         pgUUID(userID),
	})
	if err != nil {
		return false, fmt.Errorf("mark notification read: %w", err)
	}
	return affected == 1, nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	affected, err := r.queries.MarkAllNotificationsRead(ctx, pgUUID(userID))
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	return affected, nil
}

func fromRow(row db.ListNotificationsRow) notificationmodel.Notification {
	notification := notificationmodel.Notification{
		ID:                uuid.UUID(row.ID.Bytes),
		Kind:              row.Kind,
		ExchangeStatus:    interfaceString(row.ExchangeStatus),
		GivesItemTitle:    row.GivesItemTitle.String,
		ReceivesItemTitle: row.ReceivesItemTitle.String,
		SupportSubject:    row.SupportSubject.String,
		ReadAt:            optionalTime(row.ReadAt),
		CreatedAt:         row.CreatedAt.Time,
	}
	if row.ChainID.Valid {
		notification.TargetType = "exchange"
		notification.ExchangeID = uuid.UUID(row.ChainID.Bytes)
	}
	if row.SupportThreadID.Valid {
		notification.TargetType = "support"
		notification.SupportThreadID = uuid.UUID(row.SupportThreadID.Bytes)
	}

	if row.MessageID.Valid {
		messageID := uuid.UUID(row.MessageID.Bytes)
		notification.MessageID = &messageID
	}
	if row.AuthorID.Valid {
		notification.Actor = &notificationmodel.Actor{
			ID:       uuid.UUID(row.AuthorID.Bytes),
			Nickname: row.AuthorNickname.String,
			PhotoURL: optionalText(row.AuthorPhotoUrl),
		}
	}

	return notification
}

func interfaceString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}

func optionalText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}

func optionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	timestamp := value.Time
	return &timestamp
}
