package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	ratingmodel "github.com/sweetlife999/chain-of-trades-avito/internal/rating/model"
)

// Границы колонки score, а не бизнес-шкала: шкалу задаёт сервис.
const (
	minStoredScore = 1
	maxStoredScore = 5
)

var (
	ErrNotFound     = errors.New("exchange not found")
	ErrForbidden    = errors.New("not an exchange participant")
	ErrNotCompleted = errors.New("exchange is not completed")
	ErrWindowClosed = errors.New("rating window has closed")
)

type Queries interface {
	UpsertExchangeRating(context.Context, db.UpsertExchangeRatingParams) (db.UpsertExchangeRatingRow, error)
	GetExchangeRatingEligibility(
		context.Context,
		db.GetExchangeRatingEligibilityParams,
	) (db.GetExchangeRatingEligibilityRow, error)
	ListUserRatings(context.Context, db.ListUserRatingsParams) ([]db.ListUserRatingsRow, error)
}

type Repository struct {
	queries Queries
}

func New(queries Queries) *Repository {
	return &Repository{queries: queries}
}

// Upsert ставит или переписывает оценку одним запросом. Все условия — участие, статус
// обмена и срок — живут в самом запросе, поэтому счастливый путь стоит один round trip.
// Разбираться, какое из них не прошло, приходится только при отказе.
func (r *Repository) Upsert(
	ctx context.Context,
	exchangeID uuid.UUID,
	raterID uuid.UUID,
	score int32,
	comment string,
) (ratingmodel.Rating, error) {
	// Шкалу проверяет сервис, но сужение до smallint не должно держаться на вере в
	// вызывающего: за этой строкой начинается ширина колонки, и здесь она и защищается.
	if score < minStoredScore || score > maxStoredScore {
		return ratingmodel.Rating{}, fmt.Errorf("score %d is out of storable range", score)
	}

	row, err := r.queries.UpsertExchangeRating(ctx, db.UpsertExchangeRatingParams{
		ExchangeID: pgUUID(exchangeID),
		RaterID:    pgUUID(raterID),
		Score:      int16(score),
		Comment:    comment,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ratingmodel.Rating{}, r.explainRejection(ctx, exchangeID, raterID)
	}
	if err != nil {
		return ratingmodel.Rating{}, fmt.Errorf("upsert exchange rating: %w", err)
	}

	return ratingmodel.Rating{
		RatedUserID: uuid.UUID(row.RatedID.Bytes),
		Score:       int32(row.Score),
		Comment:     row.Comment,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

// Ноль строк от upsert'а не говорит, что именно не сошлось. Отдельный запрос читает
// состояние обмена и превращает отказ в конкретную ошибку.
func (r *Repository) explainRejection(
	ctx context.Context,
	exchangeID uuid.UUID,
	raterID uuid.UUID,
) error {
	eligibility, err := r.queries.GetExchangeRatingEligibility(
		ctx,
		db.GetExchangeRatingEligibilityParams{
			ExchangeID: pgUUID(exchangeID),
			RaterID:    pgUUID(raterID),
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get exchange rating eligibility: %w", err)
	}

	switch {
	case !eligibility.IsParticipant:
		return ErrForbidden
	case eligibility.ExchangeStatus != db.ChainStatusCompleted:
		return ErrNotCompleted
	case !eligibility.WindowOpen:
		return ErrWindowClosed
	}

	// Все проверки сошлись, а строки нет: партнёр по циклу не нашёлся, то есть цепочка
	// разорвана в базе. Это не отказ пользователю, а поломка данных.
	return fmt.Errorf("exchange rating rejected without a reason: %w", ErrNotFound)
}

func (r *Repository) ListForUser(
	ctx context.Context,
	ratedID uuid.UUID,
	limit int32,
	offset int32,
) ([]ratingmodel.ReceivedRating, error) {
	rows, err := r.queries.ListUserRatings(ctx, db.ListUserRatingsParams{
		RatedID:    pgUUID(ratedID),
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list user ratings: %w", err)
	}

	ratings := make([]ratingmodel.ReceivedRating, len(rows))
	for index, row := range rows {
		ratings[index] = ratingmodel.ReceivedRating{
			Score:     int32(row.Score),
			Comment:   row.Comment,
			CreatedAt: row.CreatedAt.Time,
		}
	}

	return ratings, nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}
