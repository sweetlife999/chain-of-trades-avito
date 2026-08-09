package service

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
)

type fakeRepository struct {
	create     func(context.Context, itemmodel.NewItem) (itemmodel.Item, error)
	get        func(context.Context, uuid.UUID) (itemmodel.Item, error)
	update     func(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error)
	remove     func(context.Context, uuid.UUID) error
	categories func(context.Context) ([]itemmodel.Category, error)
	hasOpen    func(context.Context, uuid.UUID) (bool, error)

	clearedPickupItem  uuid.UUID
	clearedPickupOwner uuid.UUID
	clearPickupErr     error
}

func (f *fakeRepository) ClearPickupPoint(_ context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	f.clearedPickupItem = id
	f.clearedPickupOwner = ownerID

	return f.clearPickupErr
}

func (f *fakeRepository) Create(ctx context.Context, item itemmodel.NewItem) (itemmodel.Item, error) {
	return f.create(ctx, item)
}

func (f *fakeRepository) GetByID(ctx context.Context, id uuid.UUID) (itemmodel.Item, error) {
	return f.get(ctx, id)
}

// Сервис только пробрасывает вызов, поэтому фейк ничего не хранит.
func (f *fakeRepository) ListByOwner(context.Context, uuid.UUID) ([]itemmodel.Item, error) {
	return nil, nil
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

func (f *fakeRepository) HasOpenExchange(ctx context.Context, id uuid.UUID) (bool, error) {
	if f.hasOpen == nil {
		return false, nil
	}

	return f.hasOpen(ctx, id)
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

func TestCreateStartsExchangeSearch(t *testing.T) {
	t.Parallel()

	input := validCreateInput()
	created := itemmodel.Item{ID: uuid.New(), OwnerID: input.OwnerID}
	repository := &fakeRepository{create: func(context.Context, itemmodel.NewItem) (itemmodel.Item, error) {
		return created, nil
	}}
	finder := &fakeExchangeFinder{}

	actual, err := newWithDependencies(repository, finder, log.Default()).Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if actual.ID != created.ID {
		t.Fatalf("Create() item = %+v, want %+v", actual, created)
	}
	if finder.calls != 1 {
		t.Fatalf("FindAndSave() calls = %d, want 1", finder.calls)
	}
	if finder.node.ItemID != created.ID || finder.node.OwnerID != created.OwnerID {
		t.Fatalf("FindAndSave() node = %+v, want item %s owner %s", finder.node, created.ID, created.OwnerID)
	}
}

func TestCompatibilityUpdateStartsExchangeSearch(t *testing.T) {
	t.Parallel()

	tests := map[string]UpdateInput{
		"category": {Category: stringPointer("books")},
		"wants":    {Wants: []string{"books"}},
	}

	for name, input := range tests {
		input := input
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ownerID := uuid.New()
			itemID := uuid.New()
			updated := itemmodel.Item{ID: itemID, OwnerID: ownerID}
			repository := &fakeRepository{
				get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
					return updated, nil
				},
				hasOpen: func(_ context.Context, actualID uuid.UUID) (bool, error) {
					if actualID != itemID {
						t.Fatalf("HasOpenExchange() ID = %s, want %s", actualID, itemID)
					}
					return false, nil
				},
				update: func(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error) {
					return updated, nil
				},
			}
			finder := &fakeExchangeFinder{}

			_, err := newWithDependencies(repository, finder, log.Default()).Update(
				context.Background(),
				itemID,
				ownerID,
				input,
			)
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}
			if finder.calls != 1 {
				t.Fatalf("FindAndSave() calls = %d, want 1", finder.calls)
			}
		})
	}
}

func TestPresentationUpdateDoesNotStartExchangeSearch(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	itemID := uuid.New()
	title := "New title"
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{ID: itemID, OwnerID: ownerID}, nil
		},
		hasOpen: func(context.Context, uuid.UUID) (bool, error) {
			t.Fatal("HasOpenExchange() must not be called")
			return false, nil
		},
		update: func(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error) {
			return itemmodel.Item{ID: itemID, OwnerID: ownerID}, nil
		},
	}
	finder := &fakeExchangeFinder{}

	_, err := newWithDependencies(repository, finder, log.Default()).Update(
		context.Background(),
		itemID,
		ownerID,
		UpdateInput{Title: &title},
	)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if finder.calls != 0 {
		t.Fatalf("FindAndSave() calls = %d, want 0", finder.calls)
	}
}

// Клиент присылает объявление целиком, поэтому category и wants приходят в теле даже
// тогда, когда пользователь тронул один заголовок. Такой запрос не меняет условий обмена
// и не должен упираться в запрет для вещи, занятой в незавершённом обмене.
func TestUpdateWithUnchangedCompatibilityIsAllowedInOpenExchange(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	itemID := uuid.New()
	current := itemmodel.Item{
		ID:       itemID,
		OwnerID:  ownerID,
		Category: "bikes",
		Wants:    []string{"books", "toys"},
	}
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return current, nil
		},
		hasOpen: func(context.Context, uuid.UUID) (bool, error) {
			return true, nil
		},
		update: func(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error) {
			return current, nil
		},
	}
	finder := &fakeExchangeFinder{}

	_, err := newWithDependencies(repository, finder, log.Default()).Update(
		context.Background(),
		itemID,
		ownerID,
		UpdateInput{
			Title:    stringPointer("New title"),
			Category: stringPointer("bikes"),
			// Порядок отличается от хранимого намеренно: из БД желания приходят
			// отсортированными, а от клиента — как он их отметил.
			Wants: []string{"toys", "books"},
		},
	)
	if err != nil {
		t.Fatalf("Update() error = %v, want success", err)
	}
	if finder.calls != 0 {
		t.Fatalf("FindAndSave() calls = %d, want 0", finder.calls)
	}
}

func TestSearchErrorIsLoggedWithoutBreakingCreate(t *testing.T) {
	t.Parallel()

	searchError := errors.New("database unavailable")
	input := validCreateInput()
	created := itemmodel.Item{ID: uuid.New(), OwnerID: input.OwnerID}
	repository := &fakeRepository{create: func(context.Context, itemmodel.NewItem) (itemmodel.Item, error) {
		return created, nil
	}}
	finder := &fakeExchangeFinder{err: searchError}
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)

	actual, err := newWithDependencies(repository, finder, logger).Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v, search errors must not break CRUD", err)
	}
	if actual.ID != created.ID {
		t.Fatalf("Create() item ID = %s, want %s", actual.ID, created.ID)
	}
	if !strings.Contains(logs.String(), searchError.Error()) || !strings.Contains(logs.String(), created.ID.String()) {
		t.Fatalf("search error was not logged with item ID: %q", logs.String())
	}
}

func TestCompatibilityUpdateRejectsItemInOpenExchange(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	itemID := uuid.New()
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{ID: itemID, OwnerID: ownerID}, nil
		},
		hasOpen: func(context.Context, uuid.UUID) (bool, error) {
			return true, nil
		},
		update: func(context.Context, uuid.UUID, itemmodel.Changes) (itemmodel.Item, error) {
			t.Fatal("Update() must not be called for an item in an open exchange")
			return itemmodel.Item{}, nil
		},
	}
	finder := &fakeExchangeFinder{}

	_, err := newWithDependencies(repository, finder, log.Default()).Update(
		context.Background(),
		itemID,
		ownerID,
		UpdateInput{Wants: []string{"books"}},
	)
	if !errors.Is(err, ErrItemInChain) {
		t.Fatalf("Update() error = %v, want %v", err, ErrItemInChain)
	}
	if finder.calls != 0 {
		t.Fatalf("FindAndSave() calls = %d, want 0", finder.calls)
	}
}

func TestCompatibilityUpdateCheckError(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")
	ownerID := uuid.New()
	itemID := uuid.New()
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{ID: itemID, OwnerID: ownerID}, nil
		},
		hasOpen: func(context.Context, uuid.UUID) (bool, error) {
			return false, databaseError
		},
	}

	_, err := New(repository).Update(
		context.Background(),
		itemID,
		ownerID,
		UpdateInput{Category: stringPointer("books")},
	)
	if !errors.Is(err, databaseError) {
		t.Fatalf("Update() error = %v, want wrapped %v", err, databaseError)
	}
}

type fakeExchangeFinder struct {
	err   error
	node  exchangemodel.Node
	calls int

	pickupItem  uuid.UUID
	pickupOwner uuid.UUID
	pickupPoint uuid.UUID
	pickupErr   error
}

func (f *fakeExchangeFinder) RecordItemPickup(
	_ context.Context,
	itemID uuid.UUID,
	ownerID uuid.UUID,
	pickupPointID uuid.UUID,
) error {
	f.pickupItem = itemID
	f.pickupOwner = ownerID
	f.pickupPoint = pickupPointID

	return f.pickupErr
}

func (f *fakeExchangeFinder) ScheduleSearch(
	_ context.Context,
	node exchangemodel.Node,
) error {
	f.calls++
	f.node = node
	return f.err
}

func stringPointer(value string) *string {
	return &value
}

func TestSetPickupPointDelegatesToExchange(t *testing.T) {
	t.Parallel()

	ownerID, itemID, pointID := uuid.New(), uuid.New(), uuid.New()
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{ID: itemID, OwnerID: ownerID}, nil
		},
	}
	finder := &fakeExchangeFinder{}
	service := newWithDependencies(repository, finder, log.Default())

	if err := service.SetPickupPoint(context.Background(), itemID, ownerID, pointID); err != nil {
		t.Fatalf("SetPickupPoint() error = %v", err)
	}
	if finder.pickupItem != itemID || finder.pickupOwner != ownerID || finder.pickupPoint != pointID {
		t.Fatalf(
			"RecordItemPickup(%s, %s, %s), want (%s, %s, %s)",
			finder.pickupItem, finder.pickupOwner, finder.pickupPoint, itemID, ownerID, pointID,
		)
	}
}

func TestSetPickupPointRejectsForeignItem(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{OwnerID: uuid.New()}, nil
		},
	}
	finder := &fakeExchangeFinder{}
	service := newWithDependencies(repository, finder, log.Default())

	err := service.SetPickupPoint(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("SetPickupPoint() error = %v, want ErrForbidden", err)
	}
	if finder.pickupItem != uuid.Nil {
		t.Fatal("SetPickupPoint() не должен доходить до обмена для чужой вещи")
	}
}

// Пока вещь занята в обмене, остальные участники уже рассчитывают на то, что она лежит
// в пункте: забрать её домой в этот момент нельзя.
func TestClearPickupPointRejectsItemInChain(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{OwnerID: ownerID}, nil
		},
		hasOpen: func(context.Context, uuid.UUID) (bool, error) {
			return true, nil
		},
	}

	err := New(repository).ClearPickupPoint(context.Background(), uuid.New(), ownerID)
	if !errors.Is(err, ErrItemInChain) {
		t.Fatalf("ClearPickupPoint() error = %v, want ErrItemInChain", err)
	}
	if repository.clearedPickupItem != uuid.Nil {
		t.Fatal("ClearPickupPoint() не должен доходить до repository для вещи в обмене")
	}
}

func TestClearPickupPointReturnsItemHome(t *testing.T) {
	t.Parallel()

	ownerID, itemID := uuid.New(), uuid.New()
	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (itemmodel.Item, error) {
			return itemmodel.Item{ID: itemID, OwnerID: ownerID}, nil
		},
	}

	if err := New(repository).ClearPickupPoint(context.Background(), itemID, ownerID); err != nil {
		t.Fatalf("ClearPickupPoint() error = %v", err)
	}
	if repository.clearedPickupItem != itemID || repository.clearedPickupOwner != ownerID {
		t.Fatalf(
			"ClearPickupPoint(%s, %s), want (%s, %s)",
			repository.clearedPickupItem, repository.clearedPickupOwner, itemID, ownerID,
		)
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
