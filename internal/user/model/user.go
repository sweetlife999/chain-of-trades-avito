package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID
	Nickname       string
	PasswordHash   string
	PhotoURL       *string
	Description    string
	DealsCompleted int32
	DealsBroken    int32
	Rating         *float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type NewUser struct {
	Nickname     string
	PasswordHash string
	PhotoURL     *string
	Description  string
}

type Changes struct {
	Nickname    *string
	PhotoURL    *string
	Description *string
}
