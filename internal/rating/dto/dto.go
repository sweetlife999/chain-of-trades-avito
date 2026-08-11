package dto

import (
	"time"

	ratingmodel "github.com/sweetlife999/chain-of-trades-avito/internal/rating/model"
)

// RateRequest — тело PUT. Кого оценивают, клиент не передаёт: партнёра задаёт сама
// цепочка, и передавать его значило бы позволить оценить кого-то другого.
type RateRequest struct {
	Score   int32  `json:"score"   example:"5"`
	Comment string `json:"comment" example:"Всё пришло вовремя, спасибо"`
}

type RatingResponse struct {
	RatedUserID string    `json:"rated_user_id"`
	Score       int32     `json:"score"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReceivedRatingResponse — строка публичной ленты отзывов. Ни автора, ни обмена, ни id
// самого отзыва: по каждому из них автор вычислялся бы обратно.
type ReceivedRatingResponse struct {
	Score     int32     `json:"score"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// Имя не PageResponse: такое уже носят страницы уведомлений и админского журнала,
// и swag сводит одноимённые схемы в нечитаемые имена с полным путём пакета.
type RatingsPageResponse struct {
	Ratings []ReceivedRatingResponse `json:"ratings"`
	Limit   int32                    `json:"limit"`
	Offset  int32                    `json:"offset"`
}

// Имя типа не ErrorResponse: такой уже есть у соседних модулей, и swag развёл бы
// коллизию нечитаемыми именами схем — тот же случай, что с ItemError.
type RatingError struct {
	Error string `json:"error"`
}

func FromModel(rating ratingmodel.Rating) RatingResponse {
	return RatingResponse{
		RatedUserID: rating.RatedUserID.String(),
		Score:       rating.Score,
		Comment:     rating.Comment,
		CreatedAt:   rating.CreatedAt,
		UpdatedAt:   rating.UpdatedAt,
	}
}

func FromPage(page ratingmodel.Page) RatingsPageResponse {
	ratings := make([]ReceivedRatingResponse, len(page.Ratings))
	for index, rating := range page.Ratings {
		ratings[index] = ReceivedRatingResponse{
			Score:     rating.Score,
			Comment:   rating.Comment,
			CreatedAt: rating.CreatedAt,
		}
	}

	return RatingsPageResponse{Ratings: ratings, Limit: page.Limit, Offset: page.Offset}
}
