//go:build integration

package service

import (
	"context"
	"os"
	"testing"
	"time"

	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
	"github.com/sweetlife999/chain-of-trades-avito/internal/llm"
)

// Набор проверяет именно живую 0.5B-модель: моками можно проверить JSON-контракт,
// но нельзя узнать, понимает ли она разговорное описание товара. Без OLLAMA_URL
// обычный CI этот тест пропускает.
func TestItemAssistantModel(t *testing.T) {
	url := os.Getenv("OLLAMA_URL")
	if url == "" {
		t.Skip("OLLAMA_URL is not set")
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen2.5:0.5b"
	}

	repository := fakeRepository{categories: []itemmodel.Category{
		{Slug: "electronics", Name: "Электроника"},
		{Slug: "phones", Name: "Смартфоны"},
		{Slug: "consoles", Name: "Игровые приставки"},
		{Slug: "computers", Name: "Компьютеры и комплектующие"},
		{Slug: "bikes", Name: "Велосипеды и транспорт"},
		{Slug: "sports", Name: "Спорт и отдых"},
		{Slug: "books", Name: "Книги"},
		{Slug: "clothes", Name: "Одежда и обувь"},
		{Slug: "furniture", Name: "Мебель и интерьер"},
		{Slug: "tools", Name: "Инструменты"},
		{Slug: "hobby", Name: "Хобби и творчество"},
		{Slug: "other", Name: "Прочее"},
	}}
	service := New(repository, llm.New(url, model))
	cases := []struct {
		name, input, category string
	}{
		{"bike", "красный горный велосипед, ездил два сезона, всё работает, на раме царапина", "bikes"},
		{"phone", "айфон 12 на 128 гб, батарея 84 процента, экран целый", "phones"},
		{"book", "сборник рассказов Чехова в твёрдом переплёте, страницы все на месте", "books"},
		{"clothes", "зимняя мужская куртка чёрная размер L, носил один сезон", "clothes"},
		{"tools", "аккумуляторная дрель, работает, в комплекте два аккумулятора", "tools"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			suggestion, err := service.generate(ctx, test.input)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			t.Logf("%s | %s | %s", suggestion.CategoryName, suggestion.Title, suggestion.Description)
			if suggestion.CategorySlug != test.category {
				t.Fatalf("category: got %q, want %q", suggestion.CategorySlug, test.category)
			}
		})
	}
}
