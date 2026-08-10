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
	// nil — вещь дома у владельца.
	PickupPoint *PickupPoint
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PickupPoint — пункт, в котором лежит вещь. Отдельный тип, а не pickuppoint/model:
// в карточке вещи нужны три поля, а даты жизни самого пункта к ней отношения не имеют.
type PickupPoint struct {
	ID      uuid.UUID
	Name    string
	Address string
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

// SearchCandidate — объявление из отменённого предложения, для которого после коммита
// нужно заново запланировать поиск. Полная карточка DFS не нужна.
type SearchCandidate struct {
	ItemID  uuid.UUID
	OwnerID uuid.UUID
}

type SearchVisibilityChange struct {
	Item               Item
	RecoveryCandidates []SearchCandidate
	Changed            bool
}
