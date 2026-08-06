package model

import (
	"time"

	"github.com/google/uuid"
)

// Message — строка треда обмена. У сообщения человека есть автор и текст, у события
// сделки текста нет, а автора может не быть: фразу frontend собирает по Kind.
type Message struct {
	ID        uuid.UUID
	Kind      string
	Body      *string
	Author    *ParticipantUser
	CreatedAt time.Time
}
