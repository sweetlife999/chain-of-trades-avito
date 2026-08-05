package model

import (
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Category    string
	Title       string
	Description string
	PhotoURLs   []string
	Wants       []string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type NewItem struct {
	OwnerID     uuid.UUID
	Category    string
	Title       string
	Description string
	PhotoURLs   []string
	Wants       []string
}

// Указатель или nil-срез означают «поле не меняем»: для списков это одно и то же,
// потому что пустой список запрещён и в API, и CHECK-ом в БД.
type Changes struct {
	Category    *string
	Title       *string
	Description *string
	PhotoURLs   []string
	Wants       []string
}

type Category struct {
	Slug string
	Name string
}
