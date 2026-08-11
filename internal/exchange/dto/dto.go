package dto

import (
	"time"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

type ExchangeResponse struct {
	ID           string                `json:"id"`
	Status       string                `json:"status"`
	CancelReason *string               `json:"cancel_reason" extensions:"x-nullable" enums:"proposal_declined,confirmed_broken,superseded,item_withdrawn,user_blocked,admin_cancelled,legacy"`
	Participants []ParticipantResponse `json:"participants"`
	UnreadCount  int64                 `json:"unread_count"`
	// null — оценивать нечего: обмен не завершён либо смотрит не участник.
	Rating    *ExchangeRatingResponse `json:"rating" extensions:"x-nullable"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
	ClosedAt  *time.Time              `json:"closed_at"`
}

// ExchangeRatingResponse — состояние оценки текущего пользователя в этом обмене.
// rated_user_id приходит с сервера, хотя клиент мог бы вычислить его сам по участникам:
// правило «кого оценивают» живёт в одном месте, и это база.
//
// Имя не RatingResponse: такой тип уже есть в rating/dto, и swag развёл бы коллизию
// нечитаемыми именами схем — тот же случай, что с ItemError.
type ExchangeRatingResponse struct {
	RatedUserID string    `json:"rated_user_id"`
	RateUntil   time.Time `json:"rate_until"`
	// null — оценка ещё не поставлена.
	Score   *int32 `json:"score" extensions:"x-nullable"`
	Comment string `json:"comment"`
}

type ParticipantResponse struct {
	User                  ParticipantUserResponse `json:"user"`
	GivesItem             ParticipantItemResponse `json:"gives_item"`
	ReceivesItem          ParticipantItemResponse `json:"receives_item"`
	Position              int32                   `json:"position"`
	Status                string                  `json:"status"`
	DecidedAt             *time.Time              `json:"decided_at"`
	CompletionConfirmedAt *time.Time              `json:"completion_confirmed_at"`
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
	// null — вещь ещё дома у владельца. По этому полю видно, кого из участников ждут.
	PickupPoint *ParticipantPickupPointResponse `json:"pickup_point" extensions:"x-nullable"`
}

type ParticipantPickupPointResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type CategoryResponse struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse — строка треда обмена. У сообщения участника есть author и body,
// у события сделки body пуст, а author пуст, если событие принадлежит всему обмену.
// Список kind перечислен в спеке: frontend собирает по нему фразу события, и молчаливое
// расхождение с enum chain_message_kind уже приводило к пропущенному виду события.
type MessageResponse struct {
	ID   string `json:"id"`
	Kind string `json:"kind" enums:"text,participant_accepted,participant_declined,participant_completed,participant_delivered_item,exchange_confirmed,exchange_delivering,exchange_delivered,exchange_completed,exchange_superseded,exchange_item_withdrawn"`
	// Заполнен только у kind = text.
	Body *string `json:"body" extensions:"x-nullable"`
	// Пуст у событий, которые принадлежат всему обмену, а не участнику.
	Author    *ParticipantUserResponse `json:"author" extensions:"x-nullable"`
	CreatedAt time.Time                `json:"created_at"`
}

type CreateMessageRequest struct {
	Body string `json:"body" binding:"required" minLength:"1" maxLength:"2000" example:"Могу привезти в субботу к метро, удобно?"`
}

// MarkReadRequest — id последнего сообщения, которое клиент показал пользователю.
// Именно оно, а не время запроса, задаёт границу прочитанного.
type MarkReadRequest struct {
	LastMessageID string `json:"last_message_id" example:"3f7c1b62-6f0f-4a2b-8f0e-2f2b9a1c7d10"`
}

func MessageFromModel(message exchangemodel.Message) MessageResponse {
	response := MessageResponse{
		ID:        message.ID.String(),
		Kind:      message.Kind,
		Body:      message.Body,
		CreatedAt: message.CreatedAt,
	}

	if message.Author != nil {
		response.Author = &ParticipantUserResponse{
			ID:       message.Author.ID.String(),
			Nickname: message.Author.Nickname,
			PhotoURL: message.Author.PhotoURL,
		}
	}

	return response
}

func MessagesFromModels(messages []exchangemodel.Message) []MessageResponse {
	responses := make([]MessageResponse, len(messages))
	for index, message := range messages {
		responses[index] = MessageFromModel(message)
	}

	return responses
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
			GivesItem:             itemFromModel(participant.GivesItem),
			ReceivesItem:          itemFromModel(participant.ReceivesItem),
			Position:              participant.Position,
			Status:                participant.Status,
			DecidedAt:             participant.DecidedAt,
			CompletionConfirmedAt: participant.CompletionConfirmedAt,
		}
	}

	var rating *ExchangeRatingResponse
	if exchange.Rating != nil {
		rating = &ExchangeRatingResponse{
			RatedUserID: exchange.Rating.RatedUserID.String(),
			RateUntil:   exchange.Rating.RateUntil,
			Score:       exchange.Rating.Score,
			Comment:     exchange.Rating.Comment,
		}
	}

	return ExchangeResponse{
		ID:           exchange.ID.String(),
		Status:       exchange.Status,
		CancelReason: exchange.CancelReason,
		Participants: participants,
		UnreadCount:  exchange.UnreadCount,
		Rating:       rating,
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
		PickupPoint: pickupPointFromModel(item.PickupPoint),
	}
}

func pickupPointFromModel(point *exchangemodel.ParticipantPickupPoint) *ParticipantPickupPointResponse {
	if point == nil {
		return nil
	}

	return &ParticipantPickupPointResponse{
		ID:      point.ID.String(),
		Name:    point.Name,
		Address: point.Address,
	}
}
