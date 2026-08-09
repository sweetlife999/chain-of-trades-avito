package dto

import (
	"time"

	adminexchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/model"
	exchangedto "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/dto"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

type ListResponse struct {
	Exchanges  []ExchangeResponse `json:"exchanges"`
	Pagination PaginationResponse `json:"pagination"`
}

type ExchangeResponse struct {
	ID           string                            `json:"id"`
	Status       string                            `json:"status" enums:"proposed,confirmed,delivering,delivered"`
	Participants []exchangedto.ParticipantResponse `json:"participants"`
	CreatedAt    time.Time                         `json:"created_at"`
	UpdatedAt    time.Time                         `json:"updated_at"`
}

type PaginationResponse struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
	Total  int64 `json:"total"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func FromModel(page adminexchangemodel.Page) ListResponse {
	exchanges := make([]ExchangeResponse, len(page.Exchanges))
	for index, exchange := range page.Exchanges {
		exchanges[index] = exchangeFromModel(exchange)
	}

	return ListResponse{
		Exchanges: exchanges,
		Pagination: PaginationResponse{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
	}
}

func exchangeFromModel(exchange exchangemodel.Details) ExchangeResponse {
	response := exchangedto.FromModel(exchange)
	return ExchangeResponse{
		ID:           response.ID,
		Status:       response.Status,
		Participants: response.Participants,
		CreatedAt:    response.CreatedAt,
		UpdatedAt:    response.UpdatedAt,
	}
}
