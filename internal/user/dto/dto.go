package dto

import (
	"time"

	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
)

type CreateUserRequest struct {
	Nickname    string  `json:"nickname"`
	Password    string  `json:"password"`
	PhotoURL    *string `json:"photo_url"`
	Description string  `json:"description"`
}

type UpdateUserRequest struct {
	Nickname    *string `json:"nickname"`
	PhotoURL    *string `json:"photo_url"`
	Description *string `json:"description"`
}

type UserResponse struct {
	ID             string    `json:"id"`
	Nickname       string    `json:"nickname"`
	PhotoURL       *string   `json:"photo_url"`
	Description    string    `json:"description"`
	DealsCompleted int32     `json:"deals_completed"`
	DealsBroken    int32     `json:"deals_broken"`
	Rating         *float64  `json:"rating"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func FromModel(user usermodel.User) UserResponse {
	return UserResponse{
		ID:             user.ID.String(),
		Nickname:       user.Nickname,
		PhotoURL:       user.PhotoURL,
		Description:    user.Description,
		DealsCompleted: user.DealsCompleted,
		DealsBroken:    user.DealsBroken,
		Rating:         user.Rating,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}
