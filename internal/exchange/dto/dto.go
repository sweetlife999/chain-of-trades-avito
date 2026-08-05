package dto

import (
	"time"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

type ExchangeResponse struct {
	ID           string                `json:"id"`
	Status       string                `json:"status"`
	Participants []ParticipantResponse `json:"participants"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	ClosedAt     *time.Time            `json:"closed_at"`
}

type ParticipantResponse struct {
	User         ParticipantUserResponse `json:"user"`
	GivesItem    ParticipantItemResponse `json:"gives_item"`
	ReceivesItem ParticipantItemResponse `json:"receives_item"`
	Position     int32                   `json:"position"`
	Status       string                  `json:"status"`
	DecidedAt    *time.Time              `json:"decided_at"`
}

type ParticipantUserResponse struct {
	ID       string  `json:"id"`
	Nickname string  `json:"nickname"`
	PhotoURL *string `json:"photo_url"`
}

type ParticipantItemResponse struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Status      string           `json:"status"`
	Category    CategoryResponse `json:"category"`
}

type CategoryResponse struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func FromModel(exchange exchangemodel.Details) ExchangeResponse {
	participants := make([]ParticipantResponse, len(exchange.Participants))
	for index, participant := range exchange.Participants {
		participants[index] = ParticipantResponse{
			User: ParticipantUserResponse{
				ID:       participant.User.ID.String(),
				Nickname: participant.User.Nickname,
				PhotoURL: participant.User.PhotoURL,
			},
			GivesItem:    itemFromModel(participant.GivesItem),
			ReceivesItem: itemFromModel(participant.ReceivesItem),
			Position:     participant.Position,
			Status:       participant.Status,
			DecidedAt:    participant.DecidedAt,
		}
	}

	return ExchangeResponse{
		ID:           exchange.ID.String(),
		Status:       exchange.Status,
		Participants: participants,
		CreatedAt:    exchange.CreatedAt,
		UpdatedAt:    exchange.UpdatedAt,
		ClosedAt:     exchange.ClosedAt,
	}
}

func FromModels(exchanges []exchangemodel.Details) []ExchangeResponse {
	responses := make([]ExchangeResponse, len(exchanges))
	for index, exchange := range exchanges {
		responses[index] = FromModel(exchange)
	}

	return responses
}

func itemFromModel(item exchangemodel.ParticipantItem) ParticipantItemResponse {
	return ParticipantItemResponse{
		ID:          item.ID.String(),
		Title:       item.Title,
		Description: item.Description,
		Status:      item.Status,
		Category: CategoryResponse{
			Slug: item.Category.Slug,
			Name: item.Category.Name,
		},
	}
}
