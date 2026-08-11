package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	ratingmodel "github.com/sweetlife999/chain-of-trades-avito/internal/rating/model"
	ratingrepository "github.com/sweetlife999/chain-of-trades-avito/internal/rating/repository"
)

const (
	minScore = 1
	maxScore = 5
	// Столько же, сколько у сообщения обмена: то же ограничение стоит в CHECK таблицы.
	// Проверка здесь не дублирует его зря — без неё длинный комментарий доехал бы до
	// Postgres и вернулся 500-й вместо 400-й.
	maxCommentLength = 2000

	DefaultLimit = 20
	MaxLimit     = 100
)

var (
	ErrValidation   = errors.New("validation error")
	ErrNotFound     = ratingrepository.ErrNotFound
	ErrForbidden    = ratingrepository.ErrForbidden
	ErrNotCompleted = ratingrepository.ErrNotCompleted
	ErrWindowClosed = ratingrepository.ErrWindowClosed
)

type Repository interface {
	Upsert(context.Context, uuid.UUID, uuid.UUID, int32, string) (ratingmodel.Rating, error)
	ListForUser(context.Context, uuid.UUID, int32, int32) ([]ratingmodel.ReceivedRating, error)
}

type Service struct {
	repository Repository
}

func New(repository Repository) *Service {
	return &Service{repository: repository}
}

// Rate ставит оценку партнёру по завершённому обмену. Кого именно оценивают, сервис не
// решает и не принимает от клиента: это участник, чья вещь пришла к оценивающему.
func (s *Service) Rate(
	ctx context.Context,
	exchangeID uuid.UUID,
	raterID uuid.UUID,
	score int32,
	comment string,
) (ratingmodel.Rating, error) {
	if exchangeID == uuid.Nil {
		return ratingmodel.Rating{}, validationError("exchange id is required")
	}
	if raterID == uuid.Nil {
		return ratingmodel.Rating{}, validationError("user id is required")
	}
	if score < minScore || score > maxScore {
		return ratingmodel.Rating{}, validationError(
			fmt.Sprintf("score must be between %d and %d", minScore, maxScore),
		)
	}

	comment = strings.TrimSpace(comment)
	if utf8.RuneCountInString(comment) > maxCommentLength {
		return ratingmodel.Rating{}, validationError(
			fmt.Sprintf("comment must not exceed %d characters", maxCommentLength),
		)
	}

	rating, err := s.repository.Upsert(ctx, exchangeID, raterID, score, comment)
	if err != nil {
		return ratingmodel.Rating{}, fmt.Errorf("rate exchange partner: %w", err)
	}

	return rating, nil
}

// ListForUser отдаёт полученные пользователем отзывы. Существование пользователя не
// проверяется: пустая страница вместо 404 не даёт перебирать чужие id.
func (s *Service) ListForUser(
	ctx context.Context,
	ratedID uuid.UUID,
	limit int32,
	offset int32,
) (ratingmodel.Page, error) {
	if ratedID == uuid.Nil {
		return ratingmodel.Page{}, validationError("user id is required")
	}
	if limit < 1 || limit > MaxLimit {
		return ratingmodel.Page{}, validationError(
			fmt.Sprintf("limit must be between 1 and %d", MaxLimit),
		)
	}
	if offset < 0 {
		return ratingmodel.Page{}, validationError("offset must not be negative")
	}

	ratings, err := s.repository.ListForUser(ctx, ratedID, limit, offset)
	if err != nil {
		return ratingmodel.Page{}, fmt.Errorf("list user ratings: %w", err)
	}

	return ratingmodel.Page{Ratings: ratings, Limit: limit, Offset: offset}, nil
}

type ValidationError struct{ message string }

func (e *ValidationError) Error() string { return e.message }
func (e *ValidationError) Unwrap() error { return ErrValidation }

func validationError(message string) error { return &ValidationError{message: message} }
