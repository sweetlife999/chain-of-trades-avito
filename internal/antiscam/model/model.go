package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusOpen      = "open"
	StatusResolved  = "resolved"
	StatusDismissed = "dismissed"

	DecisionConfirmed     = "confirmed"
	DecisionFalsePositive = "false_positive"
)

var Categories = []string{
	"credentials",
	"external_payment",
	"external_contact",
	"phishing",
	"pressure",
	"other",
}

type Job struct {
	ID        uuid.UUID
	MessageID uuid.UUID
	Attempts  int32
}

type Message struct {
	ID             uuid.UUID
	ExchangeID     uuid.UUID
	AuthorID       uuid.UUID
	AuthorNickname string
	Body           string
}

type ContextMessage struct {
	ID        uuid.UUID
	AuthorID  uuid.UUID
	Nickname  string
	Body      string
	CreatedAt time.Time
}

type Analysis struct {
	RuleScore       int32
	RuleHits        []string
	ModelSuspicious *bool
	ModelSeverity   *string
	Category        *string
	Reason          string
	Evidence        []string
	Risk            int32
	ModelName       string
	PromptVersion   string
	Suspicious      bool
}

type User struct {
	ID       uuid.UUID
	Nickname string
	PhotoURL *string
}

type Evidence struct {
	ID        uuid.UUID
	Body      string
	CreatedAt time.Time
}

type Case struct {
	ID                uuid.UUID
	ExchangeID        uuid.UUID
	Suspect           User
	Status            string
	Risk              int32
	Category          string
	Reason            string
	Reviewer          *User
	Decision          *string
	ResolutionComment string
	LatestEvidence    Evidence
	EvidenceCount     int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ClosedAt          *time.Time
}

type Filter struct {
	Status   string
	Category string
	MinRisk  int32
	Limit    int32
	Offset   int32
}

type Page struct {
	Cases  []Case
	Limit  int32
	Offset int32
	Total  int64
}
