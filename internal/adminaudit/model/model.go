package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	ActionReportAssigned       = "report_assigned"
	ActionReportResolved       = "report_resolved"
	ActionReportRejected       = "report_rejected"
	ActionReportMessagesViewed = "report_messages_viewed"
	ActionUserBlocked          = "user_blocked"
	ActionUserUnblocked        = "user_unblocked"
	ActionExchangeCancelled    = "exchange_cancelled"
)

var Actions = []string{
	ActionReportAssigned, ActionReportResolved, ActionReportRejected,
	ActionReportMessagesViewed, ActionUserBlocked, ActionUserUnblocked,
	ActionExchangeCancelled,
}

type UserBlockState struct {
	ID        uuid.UUID
	Nickname  string
	IsBlocked bool
	BlockedAt *time.Time
	BlockedBy *uuid.UUID
}

type Entry struct {
	ID         uuid.UUID
	AdminID    uuid.UUID
	Action     string
	TargetType string
	TargetID   uuid.UUID
	Metadata   json.RawMessage
	CreatedAt  time.Time
}

type Filter struct {
	AdminID *uuid.UUID
	Action  string
	From    *time.Time
	To      *time.Time
	Limit   int32
	Offset  int32
}

type Page struct {
	Entries []Entry
	Limit   int32
	Offset  int32
	Total   int64
}
