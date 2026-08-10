package model

import (
	"time"

	"github.com/google/uuid"
)

type Actor struct {
	ID       uuid.UUID
	Nickname string
	PhotoURL *string
}

type Notification struct {
	ID                uuid.UUID
	TargetType        string
	ExchangeID        uuid.UUID
	SupportThreadID   uuid.UUID
	MessageID         *uuid.UUID
	Kind              string
	Actor             *Actor
	ExchangeStatus    string
	GivesItemTitle    string
	ReceivesItemTitle string
	SupportSubject    string
	ReadAt            *time.Time
	CreatedAt         time.Time
}

type Filter struct {
	UnreadOnly bool
	Limit      int32
	Offset     int32
}

type Page struct {
	Notifications []Notification
	UnreadCount   int64
	Limit         int32
	Offset        int32
}
