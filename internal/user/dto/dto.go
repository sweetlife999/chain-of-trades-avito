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
	ID             string             `json:"id"`
	Nickname       string             `json:"nickname"`
	PhotoURL       *string            `json:"photo_url"`
	Description    string             `json:"description"`
	DealsCompleted int32              `json:"deals_completed"`
	DealsBroken    int32              `json:"deals_broken"`
	Experience     ExperienceResponse `json:"experience"`
	// null — оценок ещё нет, и это не ноль. Сколько их — в ratings_count.
	Rating       *float64  `json:"rating" extensions:"x-nullable"`
	RatingsCount int32     `json:"ratings_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ExperienceResponse is calculated from deals_completed and therefore does
// not require a separate database column or a backfill migration.
type ExperienceResponse struct {
	Level           int32 `json:"level"`
	MaxLevel        int32 `json:"max_level"`
	TotalXP         int64 `json:"total_xp"`
	CurrentLevelXP  int32 `json:"current_level_xp"`
	NextLevelXP     int32 `json:"next_level_xp"`
	ProgressPercent int32 `json:"progress_percent"`
	DealsToNext     int32 `json:"deals_to_next_level"`
	IsMaxLevel      bool  `json:"is_max_level"`
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
	experience := usermodel.ExperienceFromCompletedDeals(user.DealsCompleted)

	return UserResponse{
		ID:             user.ID.String(),
		Nickname:       user.Nickname,
		PhotoURL:       user.PhotoURL,
		Description:    user.Description,
		DealsCompleted: user.DealsCompleted,
		DealsBroken:    user.DealsBroken,
		Experience: ExperienceResponse{
			Level:           experience.Level,
			MaxLevel:        experience.MaxLevel,
			TotalXP:         experience.TotalXP,
			CurrentLevelXP:  experience.CurrentLevelXP,
			NextLevelXP:     experience.NextLevelXP,
			ProgressPercent: experience.ProgressPercent,
			DealsToNext:     experience.DealsToNext,
			IsMaxLevel:      experience.IsMaxLevel,
		},
		Rating:       user.Rating,
		RatingsCount: user.RatingsCount,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
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
