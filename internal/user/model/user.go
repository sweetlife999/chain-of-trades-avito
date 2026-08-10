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
	IsAdmin        bool
	IsBlocked      bool
	BlockedAt      *time.Time
	BlockedBy      *uuid.UUID
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

// BlockedUser — публичные данные пользователя в личном списке блокировок.
// Причину блокировки не храним: для MVP достаточно самой связи и времени.
type BlockedUser struct {
	ID        uuid.UUID
	Nickname  string
	PhotoURL  *string
	BlockedAt time.Time
}
