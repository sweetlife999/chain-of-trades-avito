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
