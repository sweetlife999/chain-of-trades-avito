package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
)

type fakeRepository struct {
	categories []itemmodel.Category
	err        error
}

func (f fakeRepository) ListCategories(context.Context) ([]itemmodel.Category, error) {
	return f.categories, f.err
}

type fakeGenerator struct {
	categoryAnswer string
	answer         string
	err            error
	user           string
	format         json.RawMessage
}

func (f *fakeGenerator) Generate(
	_ context.Context,
	_, user string,
	format json.RawMessage,
) (string, error) {
	f.user = user
	f.format = format
	return f.categoryAnswer, f.err
}

func (f *fakeGenerator) GenerateDetailed(
	_ context.Context,
	_, user string,
	format json.RawMessage,
) (string, error) {
	f.user = user
	f.format = format
	return f.answer, f.err
}

func TestGenerateMapsCategoryAndCleansText(t *testing.T) {
	generator := &fakeGenerator{categoryAnswer: `{"category":"electronics"}`, answer: `{
		"title":"  Плёночный\n фотоаппарат  ",
		"description":" Рабочий аппарат.   Есть чехол. "
	}`}
	service := New(fakeRepository{categories: []itemmodel.Category{
		{Slug: "clothes", Name: "Одежда"},
		{Slug: "electronics", Name: "Электроника"},
	}}, generator)

	suggestion, err := service.generate(context.Background(), "пленочный фотик рабочий с чехлом")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if suggestion.Title != "Плёночный фотоаппарат" {
		t.Fatalf("unexpected title: %q", suggestion.Title)
	}
	if suggestion.CategorySlug != "electronics" || suggestion.CategoryName != "Электроника" {
		t.Fatalf("unexpected category: %#v", suggestion)
	}
	if suggestion.Description != "Рабочий аппарат. Есть чехол." {
		t.Fatalf("unexpected description: %q", suggestion.Description)
	}
	if !strings.Contains(generator.user, `"item_text":"пленочный фотик рабочий с чехлом"`) {
		t.Fatalf("prompt does not contain encoded user input: %s", generator.user)
	}
	if len(generator.format) == 0 {
		t.Fatal("response schema is empty")
	}
}

func TestGenerateRejectsUnknownCategory(t *testing.T) {
	service := New(
		fakeRepository{categories: []itemmodel.Category{{Slug: "books", Name: "Книги"}}},
		&fakeGenerator{categoryAnswer: `{"category":"electronics"}`},
	)

	_, err := service.generate(context.Background(), "редкая вещь из домашней коллекции")
	if err == nil || !strings.Contains(err.Error(), "unknown category") {
		t.Fatalf("expected category validation error, got %v", err)
	}
}

func TestSubmitAndGetAreOwnerScoped(t *testing.T) {
	service := New(fakeRepository{}, &fakeGenerator{})
	ownerID := uuid.New()
	job, err := service.Submit(ownerID, "красный горный велосипед")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if job.Status != StatusPending {
		t.Fatalf("unexpected status: %s", job.Status)
	}
	if _, err := service.Get(uuid.New(), job.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	owned, err := service.Get(ownerID, job.ID)
	if err != nil {
		t.Fatalf("get owned: %v", err)
	}
	if owned.OwnerID != uuid.Nil || owned.input != "" {
		t.Fatal("private job fields leaked")
	}
}

func TestGetExpiresJob(t *testing.T) {
	service := New(fakeRepository{}, &fakeGenerator{})
	now := time.Date(2026, time.August, 14, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	ownerID := uuid.New()
	job, err := service.Submit(ownerID, "рабочая настольная лампа")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	now = now.Add(jobLifetime + time.Second)
	if _, err := service.Get(ownerID, job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestSubmitValidatesInput(t *testing.T) {
	service := New(fakeRepository{}, &fakeGenerator{})
	for _, input := range []string{"коротко", strings.Repeat("я", maxInputRunes+1)} {
		if _, err := service.Submit(uuid.New(), input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("input length %d: expected validation error, got %v", len([]rune(input)), err)
		}
	}
}

func TestCategoryByKeywordsUsesOnlyRepositoryCategories(t *testing.T) {
	categories := []itemmodel.Category{
		{Slug: "phones", Name: "Смартфоны"},
		{Slug: "other", Name: "Прочее"},
	}
	category, ok := categoryByKeywords("Айфон 12 с целым экраном", categories)
	if !ok || category.Slug != "phones" {
		t.Fatalf("unexpected category: %#v, %v", category, ok)
	}
	if _, ok := categoryByKeywords("горный велосипед", categories); ok {
		t.Fatal("category absent from repository must not be returned")
	}
}

func TestFallbackTitleKeepsRussianInput(t *testing.T) {
	if title := fallbackTitle("зимняя мужская куртка чёрная, размер L, один сезон"); title != "зимняя мужская куртка чёрная" {
		t.Fatalf("unexpected fallback title: %q", title)
	}
	if !containsCyrillic("куртка L") || containsCyrillic("winter jacket") {
		t.Fatal("cyrillic detection is incorrect")
	}
}
