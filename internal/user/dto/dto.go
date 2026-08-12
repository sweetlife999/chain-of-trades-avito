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

// Аватарка — ссылка, которую вернул POST /uploads; пустая строка убирает фотографию.
// Внешние http(s)-адреса тоже принимаются: с ними живут профили, заведённые раньше.
type UpdateUserRequest struct {
	Nickname    *string `json:"nickname"`
	PhotoURL    *string `json:"photo_url" example:"/uploads/8db9f3e2-8a45-4a70-b3d1-167b4f97e121.jpg"`
	Description *string `json:"description"`
}

type UserResponse struct {
	ID             string  `json:"id"`
	Nickname       string  `json:"nickname"`
	PhotoURL       *string `json:"photo_url"`
	Description    string  `json:"description"`
	DealsCompleted int32   `json:"deals_completed"`
	DealsBroken    int32   `json:"deals_broken"`
	// null — оценок ещё нет, и это не ноль. Сколько их — в ratings_count.
	Rating       *float64  `json:"rating" extensions:"x-nullable"`
	RatingsCount int32     `json:"ratings_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type BlockedUserResponse struct {
	ID        string    `json:"id"`
	Nickname  string    `json:"nickname"`
	PhotoURL  *string   `json:"photo_url"`
	BlockedAt time.Time `json:"blocked_at"`
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
		RatingsCount:   user.RatingsCount,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}

func BlockedFromModels(users []usermodel.BlockedUser) []BlockedUserResponse {
	result := make([]BlockedUserResponse, len(users))
	for index, user := range users {
		result[index] = BlockedUserResponse{
			ID:        user.ID.String(),
			Nickname:  user.Nickname,
			PhotoURL:  user.PhotoURL,
			BlockedAt: user.BlockedAt,
		}
	}

	return result
}
