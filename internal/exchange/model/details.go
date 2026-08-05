package model

import (
	"time"

	"github.com/google/uuid"
)

type Details struct {
	ID           uuid.UUID
	Status       string
	Participants []DetailsParticipant
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ClosedAt     *time.Time
}

type DetailsParticipant struct {
	User         ParticipantUser
	GivesItem    ParticipantItem
	ReceivesItem ParticipantItem
	Position     int32
	Status       string
	DecidedAt    *time.Time
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
}

type ParticipantCategory struct {
	Slug string
	Name string
}
