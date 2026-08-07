package dto

import (
	"time"

	pickuppointmodel "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/model"
)

type CreatePickupPointRequest struct {
	Name    string `json:"name"    example:"ПВЗ на Ленина"`
	Address string `json:"address" example:"Казань, улица Ленина, 10"`
}

type UpdatePickupPointRequest struct {
	Name    *string `json:"name"    example:"ПВЗ на Ленина"`
	Address *string `json:"address" example:"Казань, улица Ленина, 12"`
}

type PickupPointResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PickupPointError struct {
	Error string `json:"error"`
}

func FromModel(point pickuppointmodel.PickupPoint) PickupPointResponse {
	return PickupPointResponse{
		ID:        point.ID.String(),
		Name:      point.Name,
		Address:   point.Address,
		CreatedAt: point.CreatedAt,
		UpdatedAt: point.UpdatedAt,
	}
}

func FromModels(points []pickuppointmodel.PickupPoint) []PickupPointResponse {
	response := make([]PickupPointResponse, 0, len(points))
	for _, point := range points {
		response = append(response, FromModel(point))
	}

	return response
}
