package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
)

type fakeRepository struct {
	create     func(context.Context, itemmodel.NewItem) (itemmodel.Item, error)
	get        func(context.Context, uuid.UUID) (itemmodel.Item, error)
	update     func(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error)
	remove     func(context.Context, uuid.UUID) error
	categories func(context.Context) ([]itemmodel.Category, error)
}

func (f *fakeRepository) Create(ctx context.Context, item itemmodel.NewItem) (itemmodel.Item, error) {
	return f.create(ctx, item)
}

func (f *fakeRepository) GetByID(ctx context.Context, id uuid.UUID) (itemmodel.Item, error) {
	return f.get(ctx, id)
}

func (f *fakeRepository) Update(ctx context.Context, id uuid.UUID, changes itemmodel.Changes) (itemmodel.Item, error) {
	return f.update(ctx, id, changes)
}

func (f *fakeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return f.remove(ctx, id)
}

func (f *fakeRepository) ListCategories(ctx context.Context) ([]itemmodel.Category, error) {
	return f.categories(ctx)
}

func validCreateInput() CreateInput {
	return CreateInput{
		OwnerID:   uuid.New(),
		Category:  "bikes",
		Title:     "Велосипед",
		PhotoURLs: []string{"https://example.com/1.jpg"},
		Wants:     []string{"consoles"},
	}
}

func TestCreateCleansInput(t *testing.T) {
	t.Parallel()

	var saved itemmodel.NewItem
	repository := &fakeRepository{
		create: func(_ context.Context, item itemmodel.NewItem) (itemmodel.Item, error) {
			saved = item
			return itemmodel.Item{}, nil
		},
	}

	input := validCreateInput()
	input.Title = "  Велосипед  "
	input.Category = "  bikes  "
	input.Description = "  Почти новый  "
	input.PhotoURLs = []string{"  https://example.com/1.jpg  ", "", "   "}
	input.Wants = []string{"  Consoles ", "consoles", "PHONES", ""}

	if _, err := New(repository).Create(context.Background(), input); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if saved.Title != "Велосипед" || saved.Category != "bikes" || saved.Description != "Почти новый" {
		t.Fatalf("поля не очищены: %+v", saved)
	}
	if len(saved.PhotoURLs) != 1 || saved.PhotoURLs[0] != "https://example.com/1.jpg" {
		t.Fatalf("photo urls = %#v, want одну очищенную ссылку", saved.PhotoURLs)
	}
	// Слаги приводятся к нижнему регистру и дедуплицируются: пара (item_id, category_id)
	// в БД — первичный ключ, дубль обернулся бы конфликтом.
	if len(saved.Wants) != 2 || saved.Wants[0] != "consoles" || saved.Wants[1] != "phones" {
		t.Fatalf("wants = %#v, want [consoles phones]", saved.Wants)
	}
}

func TestCreateValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		modify func(*CreateInput)
	}{
		{name: "без фото", modify: func(i *CreateInput) { i.PhotoURLs = nil }},
		{name: "пустой список фото", modify: func(i *CreateInput) { i.PhotoURLs = []string{} }},
		{name: "фото из пробелов", modify: func(i *CreateInput) { i.PhotoURLs = []string{"   "} }},
		{name: "фото без схемы", modify: func(i *CreateInput) { i.PhotoURLs = []string{"example.com/1.jpg"} }},
		{name: "фото не http", modify: func(i *CreateInput) { i.PhotoURLs = []string{"ftp://example.com/1.jpg"} }},
		{name: "фото без хоста", modify: func(i *CreateInput) { i.PhotoURLs = []string{"https:///1.jpg"} }},
		{name: "больше десяти фото", modify: func(i *CreateInput) { i.PhotoURLs = photos(11) }},
		{name: "без желаний", modify: func(i *CreateInput) { i.Wants = nil }},
		{name: "пустой список желаний", modify: func(i *CreateInput) { i.Wants = []string{} }},
		{name: "желания из пробелов", modify: func(i *CreateInput) { i.Wants = []string{" ", ""} }},
		{name: "больше десяти желаний", modify: func(i *CreateInput) { i.Wants = slugs(11) }},
		{name: "пустой заголовок", modify: func(i *CreateInput) { i.Title = "   " }},
		{name: "слишком длинный заголовок", modify: func(i *CreateInput) { i.Title = strings.Repeat("я", 121) }},
		{name: "без категории", modify: func(i *CreateInput) { i.Category = "  " }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeRepository{
				create: func(context.Context, itemmodel.NewItem) (itemmodel.Item, error) {
					t.Fatal("Create() не должен доходить до repository")
					return itemmodel.Item{}, nil
				},
			}

			input := validCreateInput()
			test.modify(&input)

			if _, err := New(repository).Create(context.Background(), input); !errors.Is(err, ErrValidation) {
				t.Fatalf("error = %v, want ErrValidation", err)
			}
		})
	}
}

func TestCreateAcceptsTenPhotosAndWants(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{
		create: func(context.Context, itemmodel.NewItem) (itemmodel.Item, error) {
			return itemmodel.Item{}, nil
		},
	}

	input := validCreateInput()
	input.PhotoURLs = photos(10)
	input.Wants = slugs(10)

	if _, err := New(repository).Create(context.Background(), input); err != nil {
		t.Fatalf("Create() с десятью фото и желаниями error = %v", err)
	}
}

func TestUpdateChangesOnlyProvidedFields(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	itemID := uuid.New()
	title := "  Новый заголовок  "

	var applied itemmodel.Changes
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{ID: itemID, OwnerID: ownerID}, nil
		},
		update: func(_ context.Context, _ uuid.UUID, changes itemmodel.Changes) (itemmodel.Item, error) {
			applied = changes
			return itemmodel.Item{ID: itemID}, nil
		},
	}

	_, err := New(repository).Update(context.Background(), itemID, ownerID, UpdateInput{
		Title: &title,
		Wants: []string{" Books ", "books"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if applied.Title == nil || *applied.Title != "Новый заголовок" {
		t.Fatalf("title = %#v", applied.Title)
	}
	if len(applied.Wants) != 1 || applied.Wants[0] != "books" {
		t.Fatalf("wants = %#v, want [books]", applied.Wants)
	}
	if applied.PhotoURLs != nil || applied.Category != nil || applied.Description != nil {
		t.Fatalf("непереданные поля попали в изменения: %+v", applied)
	}
}

func TestUpdateValidation(t *testing.T) {
	t.Parallel()

	empty := ""
	tests := []struct {
		name  string
		input UpdateInput
	}{
		{name: "ни одного поля", input: UpdateInput{}},
		{name: "удаление всех фото", input: UpdateInput{PhotoURLs: []string{}}},
		{name: "фото из пробелов", input: UpdateInput{PhotoURLs: []string{"  "}}},
		{name: "битая ссылка на фото", input: UpdateInput{PhotoURLs: []string{"/photo.jpg"}}},
		{name: "больше десяти фото", input: UpdateInput{PhotoURLs: photos(11)}},
		{name: "удаление всех желаний", input: UpdateInput{Wants: []string{}}},
		{name: "больше десяти желаний", input: UpdateInput{Wants: slugs(11)}},
		{name: "пустой заголовок", input: UpdateInput{Title: &empty}},
		{name: "пустая категория", input: UpdateInput{Category: &empty}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ownerID := uuid.New()
			repository := &fakeRepository{
				get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
					return itemmodel.Item{OwnerID: ownerID}, nil
				},
				update: func(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error) {
					t.Fatal("Update() не должен доходить до repository")
					return itemmodel.Item{}, nil
				},
			}

			_, err := New(repository).Update(context.Background(), uuid.New(), ownerID, test.input)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("error = %v, want ErrValidation", err)
			}
		})
	}
}

func TestUpdateAndDeleteRejectSomeoneElsesItem(t *testing.T) {
	t.Parallel()

	title := "Чужая вещь"
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{OwnerID: uuid.New()}, nil
		},
		update: func(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error) {
			t.Fatal("Update() не должен доходить до repository")
			return itemmodel.Item{}, nil
		},
		remove: func(context.Context, uuid.UUID) error {
			t.Fatal("Delete() не должен доходить до repository")
			return nil
		},
	}

	service := New(repository)

	if _, err := service.Update(context.Background(), uuid.New(), uuid.New(), UpdateInput{Title: &title}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
	if err := service.Delete(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Delete() error = %v, want ErrForbidden", err)
	}
}

func TestDeleteRemovesOwnItem(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	itemID := uuid.New()

	deleted := uuid.Nil
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{ID: itemID, OwnerID: ownerID}, nil
		},
		remove: func(_ context.Context, id uuid.UUID) error {
			deleted = id
			return nil
		},
	}

	if err := New(repository).Delete(context.Background(), itemID, ownerID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted != itemID {
		t.Fatalf("удалили %v, а просили %v", deleted, itemID)
	}
}

func photos(n int) []string {
	list := make([]string, 0, n)
	for i := range n {
		list = append(list, "https://example.com/"+string(rune('a'+i))+".jpg")
	}

	return list
}

func slugs(n int) []string {
	list := make([]string, 0, n)
	for i := range n {
		list = append(list, "category-"+string(rune('a'+i)))
	}

	return list
}
