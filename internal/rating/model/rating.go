package model

import (
	"time"

	"github.com/google/uuid"
)

// Rating — оценка, которую поставил текущий пользователь. Кого он оценил, решает не он:
// партнёр выводится из самой цепочки, поэтому в модели это результат, а не аргумент.
type Rating struct {
	RatedUserID uuid.UUID
	Score       int32
	Comment     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ReceivedRating — отзыв в публичной ленте профиля. Автора здесь нет и не будет:
// анонимность ленты держится тем, что его негде взять, а не тем, что его не показывают.
type ReceivedRating struct {
	Score     int32
	Comment   string
	CreatedAt time.Time
}

type Page struct {
	Ratings []ReceivedRating
	Limit   int32
	Offset  int32
}
