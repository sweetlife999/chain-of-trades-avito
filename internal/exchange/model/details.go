package model

import (
	"time"

	"github.com/google/uuid"
)

type Details struct {
	ID           uuid.UUID
	Status       string
	CancelReason *string
	Participants []DetailsParticipant
	UnreadCount  int64
	// nil, пока обмен не завершён или смотрит посторонний. Оценка — свойство «меня» в
	// обмене, поэтому она одна на весь ответ, а не поле участника.
	Rating    *DetailsRating
	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  *time.Time
}

// DetailsRating — кого текущий пользователь должен оценить, до какого момента и что он
// уже поставил. Партнёра считает база: повторять это правило на клиенте значило бы
// завести ему второй источник правды.
type DetailsRating struct {
	RatedUserID uuid.UUID
	RateUntil   time.Time
	// nil — оценка ещё не поставлена.
	Score   *int32
	Comment string
}

type DetailsParticipant struct {
	User                  ParticipantUser
	GivesItem             ParticipantItem
	ReceivesItem          ParticipantItem
	Position              int32
	Status                string
	DecidedAt             *time.Time
	CompletionConfirmedAt *time.Time
}

type ParticipantUser struct {
	ID       uuid.UUID
	Nickname string
	PhotoURL *string
}

type ParticipantItem struct {
	ID          uuid.UUID
	Title       string
	Description string
	Status      string
	Category    ParticipantCategory
	// nil — вещь ещё дома у владельца. По этому полю участник видит, кого ждут.
	PickupPoint *ParticipantPickupPoint
}

type ParticipantPickupPoint struct {
	ID      uuid.UUID
	Name    string
	Address string
}

type ParticipantCategory struct {
	Slug string
	Name string
}
