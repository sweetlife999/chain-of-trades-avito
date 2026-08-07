package model

import (
	"time"

	"github.com/google/uuid"
)

type PickupPoint struct {
	ID        uuid.UUID
	Name      string
	Address   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewPickupPoint struct {
	Name    string
	Address string
}

type Changes struct {
	Name    *string
	Address *string
}
