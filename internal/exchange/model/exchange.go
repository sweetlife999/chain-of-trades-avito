package model

import "github.com/google/uuid"

type Exchange struct {
	Participants []Participant
}

type Participant struct {
	UserID         uuid.UUID
	GivesItemID    uuid.UUID
	ReceivesItemID uuid.UUID
	Position       int32
}
