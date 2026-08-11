package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	ratingmodel "github.com/sweetlife999/chain-of-trades-avito/internal/rating/model"
)

type fakeRepository struct {
	rating      ratingmodel.Rating
	received    []ratingmodel.ReceivedRating
	err         error
	called      bool
	lastScore   int32
	lastComment string
	lastLimit   int32
	lastOffset  int32
}

func (f *fakeRepository) Upsert(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	score int32,
	comment string,
) (ratingmodel.Rating, error) {
	f.called = true
	f.lastScore = score
	f.lastComment = comment
	return f.rating, f.err
}

func (f *fakeRepository) ListForUser(
	_ context.Context,
	_ uuid.UUID,
	limit int32,
	offset int32,
) ([]ratingmodel.ReceivedRating, error) {
	f.called = true
	f.lastLimit = limit
	f.lastOffset = offset
	return f.received, f.err
}

func TestRateRejectsScoreOutsideScale(t *testing.T) {
	t.Parallel()

	for _, score := range []int32{0, -1, 6, 100} {
		repository := &fakeRepository{}
		_, err := New(repository).Rate(context.Background(), uuid.New(), uuid.New(), score, "")
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("score %d: ожидали ErrValidation, получили %v", score, err)
		}
		if repository.called {
			t.Fatalf("score %d: запрос ушёл в базу, хотя балл вне шкалы", score)
		}
	}
}

// Длину проверяет и CHECK в таблице, но нарушение доехало бы туда как 23514 и вернулось
// пятисоткой. Поэтому граница обязана срабатывать здесь.
func TestRateRejectsTooLongComment(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	comment := strings.Repeat("я", maxCommentLength+1)

	_, err := New(repository).Rate(context.Background(), uuid.New(), uuid.New(), 5, comment)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ожидали ErrValidation, получили %v", err)
	}
	if repository.called {
		t.Fatal("слишком длинный комментарий ушёл в базу")
	}
}

// Ровно на границе — считаем символы, а не байты: в CHECK стоит char_length.
func TestRateAcceptsCommentAtLimit(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	comment := strings.Repeat("я", maxCommentLength)

	if _, err := New(repository).Rate(context.Background(), uuid.New(), uuid.New(), 5, comment); err != nil {
		t.Fatalf("комментарий длиной ровно %d символов отвергнут: %v", maxCommentLength, err)
	}
	if repository.lastComment != comment {
		t.Fatal("комментарий доехал до репозитория изменённым")
	}
}

func TestRateTrimsComment(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}

	if _, err := New(repository).Rate(
		context.Background(), uuid.New(), uuid.New(), 4, "  всё вовремя  ",
	); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if repository.lastComment != "всё вовремя" {
		t.Fatalf("ожидали обрезанный комментарий, получили %q", repository.lastComment)
	}
	if repository.lastScore != 4 {
		t.Fatalf("балл доехал искажённым: %d", repository.lastScore)
	}
}

func TestRateRequiresIdentifiers(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	if _, err := New(repository).Rate(
		context.Background(), uuid.Nil, uuid.New(), 5, "",
	); !errors.Is(err, ErrValidation) {
		t.Fatalf("пустой обмен: ожидали ErrValidation, получили %v", err)
	}
	if _, err := New(repository).Rate(
		context.Background(), uuid.New(), uuid.Nil, 5, "",
	); !errors.Is(err, ErrValidation) {
		t.Fatalf("пустой пользователь: ожидали ErrValidation, получили %v", err)
	}
}

// Отказ базы — участие, статус, срок — сервис не переписывает: handler различает их по
// сентинелам, и подмена любого из них превратилась бы в неверный код ответа.
func TestRatePassesRepositoryErrorThrough(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{err: ErrWindowClosed}

	_, err := New(repository).Rate(context.Background(), uuid.New(), uuid.New(), 5, "")
	if !errors.Is(err, ErrWindowClosed) {
		t.Fatalf("ожидали ErrWindowClosed, получили %v", err)
	}
}

func TestListForUserValidatesPage(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	service := New(repository)

	for _, limit := range []int32{0, -1, MaxLimit + 1} {
		if _, err := service.ListForUser(
			context.Background(), uuid.New(), limit, 0,
		); !errors.Is(err, ErrValidation) {
			t.Fatalf("limit %d: ожидали ErrValidation, получили %v", limit, err)
		}
	}
	if _, err := service.ListForUser(
		context.Background(), uuid.New(), DefaultLimit, -1,
	); !errors.Is(err, ErrValidation) {
		t.Fatalf("отрицательный offset: ожидали ErrValidation, получили %v", err)
	}
	if repository.called {
		t.Fatal("некорректная страница ушла в базу")
	}
}

func TestListForUserReturnsPage(t *testing.T) {
	t.Parallel()

	want := []ratingmodel.ReceivedRating{{Score: 5, Comment: "спасибо"}}
	repository := &fakeRepository{received: want}

	page, err := New(repository).ListForUser(context.Background(), uuid.New(), 10, 20)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(page.Ratings) != 1 || page.Ratings[0].Score != 5 {
		t.Fatalf("страница вернулась не той: %#v", page.Ratings)
	}
	if page.Limit != 10 || page.Offset != 20 {
		t.Fatalf("границы страницы потерялись: limit=%d offset=%d", page.Limit, page.Offset)
	}
	if repository.lastLimit != 10 || repository.lastOffset != 20 {
		t.Fatalf("границы доехали до базы искажёнными: limit=%d offset=%d",
			repository.lastLimit, repository.lastOffset)
	}
}
