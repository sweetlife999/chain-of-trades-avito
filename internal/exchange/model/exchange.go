package model

import (
	"sort"
	"strings"

	"github.com/google/uuid"
)

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

// Signature возвращает каноническую подпись направленного цикла. Порядок строк
// участников не влияет на результат, а направление каждой передачи сохраняется.
func (e Exchange) Signature() string {
	transfers := make([]string, len(e.Participants))
	for index, participant := range e.Participants {
		transfers[index] = participant.GivesItemID.String() + ">" + participant.ReceivesItemID.String()
	}

	sort.Strings(transfers)
	return strings.Join(transfers, "|")
}

// CompositionKey identifies the set of items independently of traversal start
// and transfer direction. Signature still describes the exact directed cycle.
func (e Exchange) CompositionKey() string {
	items := make([]string, len(e.Participants))
	for index, participant := range e.Participants {
		items[index] = participant.GivesItemID.String()
	}

	sort.Strings(items)
	return strings.Join(items, "|")
}
