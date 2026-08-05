package model

import "github.com/google/uuid"

// Node — минимальное представление объявления, нужное алгоритму поиска обмена.
// Остальные данные объявления DFS не использует.
type Node struct {
	ItemID  uuid.UUID
	OwnerID uuid.UUID
}
