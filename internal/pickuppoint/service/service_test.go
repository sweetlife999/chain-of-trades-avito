package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	pickuppointmodel "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/model"
)

type fakeRepository struct {
	create func(context.Context, pickuppointmodel.NewPickupPoint) (pickuppointmodel.PickupPoint, error)
	get    func(context.Context, uuid.UUID) (pickuppointmodel.PickupPoint, error)
	list   func(context.Context) ([]pickuppointmodel.PickupPoint, error)
	update func(context.Context, uuid.UUID, pickuppointmodel.Changes) (pickuppointmodel.PickupPoint, error)
	delete func(context.Context, uuid.UUID) error
}

func (f *fakeRepository) Create(ctx context.Context, point pickuppointmodel.NewPickupPoint) (pickuppointmodel.PickupPoint, error) {
	return f.create(ctx, point)
}

func (f *fakeRepository) GetByID(ctx context.Context, id uuid.UUID) (pickuppointmodel.PickupPoint, error) {
	return f.get(ctx, id)
}

func (f *fakeRepository) List(ctx context.Context) ([]pickuppointmodel.PickupPoint, error) {
	return f.list(ctx)
}

func (f *fakeRepository) Update(ctx context.Context, id uuid.UUID, changes pickuppointmodel.Changes) (pickuppointmodel.PickupPoint, error) {
	return f.update(ctx, id, changes)
}

func (f *fakeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return f.delete(ctx, id)
}

func TestCreateCleansInput(t *testing.T) {
	t.Parallel()

	var saved pickuppointmodel.NewPickupPoint
	repository := &fakeRepository{
		create: func(_ context.Context, point pickuppointmodel.NewPickupPoint) (pickuppointmodel.PickupPoint, error) {
			saved = point
			return pickuppointmodel.PickupPoint{ID: uuid.New(), Name: point.Name, Address: point.Address}, nil
		},
	}

	created, err := New(repository).Create(context.Background(), CreateInput{
		Name:    "  ПВЗ на Ленина  ",
		Address: "  ул. Ленина, 10  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if saved.Name != "ПВЗ на Ленина" || saved.Address != "ул. Ленина, 10" {
		t.Fatalf("saved = %+v, want trimmed fields", saved)
	}
	if created.Name != saved.Name || created.Address != saved.Address {
		t.Fatalf("created = %+v, want repository result", created)
	}
}

func TestCreateValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input CreateInput
	}{
		{name: "empty name", input: CreateInput{Name: "  ", Address: "address"}},
		{name: "long name", input: CreateInput{Name: strings.Repeat("я", maxNameLength+1), Address: "address"}},
		{name: "empty address", input: CreateInput{Name: "name", Address: "  "}},
		{name: "long address", input: CreateInput{Name: "name", Address: strings.Repeat("я", maxAddressLength+1)}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeRepository{
				create: func(context.Context, pickuppointmodel.NewPickupPoint) (pickuppointmodel.PickupPoint, error) {
					t.Fatal("Create() must not reach repository")
					return pickuppointmodel.PickupPoint{}, nil
				},
			}
			if _, err := New(repository).Create(context.Background(), test.input); !errors.Is(err, ErrValidation) {
				t.Fatalf("error = %v, want ErrValidation", err)
			}
		})
	}
}

func TestUpdateChangesOnlyProvidedFields(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	name := "  Новый ПВЗ  "
	var applied pickuppointmodel.Changes
	repository := &fakeRepository{
		update: func(_ context.Context, actualID uuid.UUID, changes pickuppointmodel.Changes) (pickuppointmodel.PickupPoint, error) {
			if actualID != id {
				t.Fatalf("id = %v, want %v", actualID, id)
			}
			applied = changes
			return pickuppointmodel.PickupPoint{ID: id, Name: *changes.Name}, nil
		},
	}

	updated, err := New(repository).Update(context.Background(), id, UpdateInput{Name: &name})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if applied.Name == nil || *applied.Name != "Новый ПВЗ" {
		t.Fatalf("name = %#v, want trimmed value", applied.Name)
	}
	if applied.Address != nil {
		t.Fatalf("address = %#v, want nil", applied.Address)
	}
	if updated.ID != id {
		t.Fatalf("updated id = %v, want %v", updated.ID, id)
	}
}

func TestUpdateValidation(t *testing.T) {
	t.Parallel()

	empty := "  "
	longName := strings.Repeat("я", maxNameLength+1)
	longAddress := strings.Repeat("я", maxAddressLength+1)
	tests := []struct {
		name  string
		input UpdateInput
	}{
		{name: "no fields", input: UpdateInput{}},
		{name: "empty name", input: UpdateInput{Name: &empty}},
		{name: "long name", input: UpdateInput{Name: &longName}},
		{name: "empty address", input: UpdateInput{Address: &empty}},
		{name: "long address", input: UpdateInput{Address: &longAddress}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeRepository{
				update: func(context.Context, uuid.UUID, pickuppointmodel.Changes) (pickuppointmodel.PickupPoint, error) {
					t.Fatal("Update() must not reach repository")
					return pickuppointmodel.PickupPoint{}, nil
				},
			}
			if _, err := New(repository).Update(context.Background(), uuid.New(), test.input); !errors.Is(err, ErrValidation) {
				t.Fatalf("error = %v, want ErrValidation", err)
			}
		})
	}
}

func TestServicePreservesRepositoryErrors(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{
		get: func(context.Context, uuid.UUID) (pickuppointmodel.PickupPoint, error) {
			return pickuppointmodel.PickupPoint{}, ErrNotFound
		},
		delete: func(context.Context, uuid.UUID) error {
			return ErrInUse
		},
	}
	service := New(repository)

	if _, err := service.GetByID(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
	if err := service.Delete(context.Background(), uuid.New()); !errors.Is(err, ErrInUse) {
		t.Fatalf("Delete() error = %v, want ErrInUse", err)
	}
}
