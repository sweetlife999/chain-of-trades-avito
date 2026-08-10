package model

import (
	"time"

	"github.com/google/uuid"
)

// Report — жалоба на сообщение треда. Автор сообщения и обмен здесь не хранятся:
// они выводятся джойном к chain_messages, см. миграцию 00012_reports.
type Report struct {
	ID        uuid.UUID
	MessageID uuid.UUID
	Reason    string
	Comment   string
	Status    string
	CreatedAt time.Time
}

// NewReport — то, что приходит от пользователя. Status и разбирающего ставит база.
type NewReport struct {
	ReporterID uuid.UUID
	MessageID  uuid.UUID
	Reason     string
	Comment    string
}

// Target — всё, что нужно знать о сообщении, чтобы решить, принимать ли жалобу.
// AuthorID пустой у событий обмена: у них автора нет.
type Target struct {
	Kind          string
	AuthorID      uuid.UUID
	IsParticipant bool
}

// AdminUser — минимальный профиль, который нужен модератору в карточке жалобы.
// Пароль, описание и внутренняя статистика пользователя сюда намеренно не попадают.
type AdminUser struct {
	ID       uuid.UUID
	Nickname string
	PhotoURL *string
}

type ReportedMessage struct {
	ID        uuid.UUID
	Body      string
	CreatedAt time.Time
}

type ReportExchange struct {
	ID     uuid.UUID
	Status string
}

// AdminReport объединяет саму жалобу с данными, которые выводятся через связи:
// жалобщиком, автором сообщения, сообщением и обменом.
type AdminReport struct {
	ID                uuid.UUID
	Reason            string
	Comment           string
	Status            string
	Reporter          AdminUser
	Offender          AdminUser
	Message           ReportedMessage
	Exchange          ReportExchange
	Assignee          *AdminUser
	CreatedAt         time.Time
	AssignedAt        *time.Time
	ClosedAt          *time.Time
	ResolutionComment string
}

type AdminFilter struct {
	Status     string
	Reason     string
	AssigneeID *uuid.UUID
	Limit      int32
	Offset     int32
}

type AdminPage struct {
	Reports []AdminReport
	Limit   int32
	Offset  int32
	Total   int64
}
