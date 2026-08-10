package dto

import (
	"time"

	"github.com/google/uuid"

	notificationmodel "github.com/sweetlife999/chain-of-trades-avito/internal/notification/model"
)

type ActorResponse struct {
	ID       string  `json:"id"`
	Nickname string  `json:"nickname"`
	PhotoURL *string `json:"photo_url"`
}

type NotificationResponse struct {
	ID                string         `json:"id"`
	Kind              string         `json:"kind"`
	TargetType        string         `json:"target_type"`
	ExchangeID        *string        `json:"exchange_id"`
	SupportThreadID   *string        `json:"support_thread_id"`
	MessageID         *string        `json:"message_id"`
	Actor             *ActorResponse `json:"actor"`
	ExchangeStatus    string         `json:"exchange_status"`
	GivesItemTitle    string         `json:"gives_item_title"`
	ReceivesItemTitle string         `json:"receives_item_title"`
	SupportSubject    string         `json:"support_subject"`
	IsRead            bool           `json:"is_read"`
	ReadAt            *time.Time     `json:"read_at"`
	CreatedAt         time.Time      `json:"created_at"`
}

type PageResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	UnreadCount   int64                  `json:"unread_count"`
	Limit         int32                  `json:"limit"`
	Offset        int32                  `json:"offset"`
}

type MarkAllResponse struct {
	MarkedCount int64 `json:"marked_count"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func FromPage(page notificationmodel.Page) PageResponse {
	notifications := make([]NotificationResponse, len(page.Notifications))
	for index, notification := range page.Notifications {
		notifications[index] = FromModel(notification)
	}
	return PageResponse{
		Notifications: notifications,
		UnreadCount:   page.UnreadCount,
		Limit:         page.Limit,
		Offset:        page.Offset,
	}
}

func FromModel(notification notificationmodel.Notification) NotificationResponse {
	response := NotificationResponse{
		ID:                notification.ID.String(),
		Kind:              notification.Kind,
		TargetType:        notification.TargetType,
		ExchangeStatus:    notification.ExchangeStatus,
		GivesItemTitle:    notification.GivesItemTitle,
		ReceivesItemTitle: notification.ReceivesItemTitle,
		SupportSubject:    notification.SupportSubject,
		IsRead:            notification.ReadAt != nil,
		ReadAt:            notification.ReadAt,
		CreatedAt:         notification.CreatedAt,
	}
	if notification.ExchangeID != uuid.Nil {
		exchangeID := notification.ExchangeID.String()
		response.ExchangeID = &exchangeID
	}
	if notification.SupportThreadID != uuid.Nil {
		threadID := notification.SupportThreadID.String()
		response.SupportThreadID = &threadID
	}
	if notification.MessageID != nil {
		messageID := notification.MessageID.String()
		response.MessageID = &messageID
	}
	if notification.Actor != nil {
		response.Actor = &ActorResponse{
			ID:       notification.Actor.ID.String(),
			Nickname: notification.Actor.Nickname,
			PhotoURL: notification.Actor.PhotoURL,
		}
	}
	return response
}
