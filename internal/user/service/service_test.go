package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
)

type fakeRepository struct {
	create func(context.Context, usermodel.NewUser) (usermodel.User, error)
	get    func(context.Context, uuid.UUID) (usermodel.User, error)
	update func(context.Context, uuid.UUID, usermodel.Changes) (usermodel.User, error)
}

func (f *fakeRepository) Create(ctx context.Context, user usermodel.NewUser) (usermodel.User, error) {
	return f.create(ctx, user)
}

func (f *fakeRepository) GetByID(ctx context.Context, id uuid.UUID) (usermodel.User, error) {
	return f.get(ctx, id)
}

func (f *fakeRepository) Update(ctx context.Context, id uuid.UUID, changes usermodel.Changes) (usermodel.User, error) {
	return f.update(ctx, id, changes)
}

func TestCreateCleansInputAndHashesPassword(t *testing.T) {
	t.Parallel()

	photoURL := "  https://example.com/photo.png  "
	repository := &fakeRepository{
		create: func(_ context.Context, user usermodel.NewUser) (usermodel.User, error) {
			if user.Nickname != "Samir" {
				t.Fatalf("nickname was not cleaned: %q", user.Nickname)
			}
			if user.Description != "description" {
				t.Fatalf("description was not cleaned: %q", user.Description)
			}
			if user.PhotoURL == nil || *user.PhotoURL != "https://example.com/photo.png" {
				t.Fatalf("photo URL was not cleaned: %#v", user.PhotoURL)
			}
			if user.PasswordHash == "password123" {
				t.Fatal("plain password was passed to repository")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123")); err != nil {
				t.Fatalf("password hash does not match password: %v", err)
			}
			return usermodel.User{Nickname: user.Nickname}, nil
		},
	}

	service := newWithCost(repository, bcrypt.MinCost)
	_, err := service.Create(context.Background(), CreateInput{
		Nickname:    "  Samir  ",
		Password:    "password123",
		PhotoURL:    &photoURL,
		Description: "  description  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input CreateInput
	}{
		{name: "short nickname", input: CreateInput{Nickname: "ab", Password: "password123"}},
		{name: "short password", input: CreateInput{Nickname: "Samir", Password: "short"}},
		{name: "long password", input: CreateInput{Nickname: "Samir", Password: string(make([]byte, 73))}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := newWithCost(&fakeRepository{}, bcrypt.MinCost)
			_, err := service.Create(context.Background(), test.input)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Create() error = %v, want validation error", err)
			}
		})
	}
}

func TestUpdateCleansFields(t *testing.T) {
	t.Parallel()

	nickname := "  NewName  "
	description := "  new description  "
	id := uuid.New()

	repository := &fakeRepository{
		update: func(_ context.Context, actualID uuid.UUID, changes usermodel.Changes) (usermodel.User, error) {
			if actualID != id {
				t.Fatalf("Update() id = %v, want %v", actualID, id)
			}
			if changes.Nickname == nil || *changes.Nickname != "NewName" {
				t.Fatalf("nickname was not cleaned: %#v", changes.Nickname)
			}
			if changes.Description == nil || *changes.Description != "new description" {
				t.Fatalf("description was not cleaned: %#v", changes.Description)
			}
			return usermodel.User{ID: id, Nickname: *changes.Nickname}, nil
		},
	}

	service := newWithCost(repository, bcrypt.MinCost)
	_, err := service.Update(context.Background(), id, UpdateInput{
		Nickname:    &nickname,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
}

func TestUpdateRejectsEmptyChanges(t *testing.T) {
	t.Parallel()

	service := newWithCost(&fakeRepository{}, bcrypt.MinCost)
	_, err := service.Update(context.Background(), uuid.New(), UpdateInput{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Update() error = %v, want validation error", err)
	}
}
