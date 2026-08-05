package model

import "github.com/google/uuid"

type Exchange struct {
	Participants []Participant
}

type SearchResult struct {
	ExchangeID uuid.UUID
	Found      bool
}

type Participant struct {
	UserID         uuid.UUID
	GivesItemID    uuid.UUID
	ReceivesItemID uuid.UUID
	Position       int32
}
