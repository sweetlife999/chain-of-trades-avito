package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
)

type fakeRepository struct {
	getByNickname func(context.Context, string) (usermodel.User, error)
	getByID       func(context.Context, uuid.UUID) (usermodel.User, error)
}

func (f *fakeRepository) GetByNickname(ctx context.Context, nickname string) (usermodel.User, error) {
	return f.getByNickname(ctx, nickname)
}

func (f *fakeRepository) GetByID(ctx context.Context, id uuid.UUID) (usermodel.User, error) {
	return f.getByID(ctx, id)
}

type fakeTokenIssuer struct {
	generate func(uuid.UUID) (string, time.Time, error)
}

func (f *fakeTokenIssuer) Generate(userID uuid.UUID) (string, time.Time, error) {
	return f.generate(userID)
}

func TestLoginChecksPasswordAndCreatesToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}
	wantExpiry := time.Now().Add(12 * time.Hour)

	repository := &fakeRepository{
		getByNickname: func(_ context.Context, nickname string) (usermodel.User, error) {
			if nickname != "Samir" {
				t.Fatalf("GetByNickname() nickname = %q, want Samir", nickname)
			}
			return usermodel.User{
				ID:           userID,
				Nickname:     "Samir",
				PasswordHash: string(passwordHash),
			}, nil
		},
	}
	tokens := &fakeTokenIssuer{
		generate: func(actualID uuid.UUID) (string, time.Time, error) {
			if actualID != userID {
				t.Fatalf("Generate() id = %v, want %v", actualID, userID)
			}
			return "signed-token", wantExpiry, nil
		},
	}

	result, err := New(repository, tokens).Login(context.Background(), LoginInput{
		Nickname: "  Samir  ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.ID != userID || result.Token != "signed-token" || !result.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("Login() result = %#v", result)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	t.Parallel()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}

	tests := []struct {
		name       string
		repository *fakeRepository
	}{
		{
			name: "unknown nickname",
			repository: &fakeRepository{
				getByNickname: func(context.Context, string) (usermodel.User, error) {
					return usermodel.User{}, userrepository.ErrNotFound
				},
			},
		},
		{
			name: "wrong password",
			repository: &fakeRepository{
				getByNickname: func(context.Context, string) (usermodel.User, error) {
					return usermodel.User{PasswordHash: string(passwordHash)}, nil
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := New(test.repository, &fakeTokenIssuer{})
			_, err := service.Login(context.Background(), LoginInput{
				Nickname: "Samir",
				Password: "wrong-password",
			})
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestLoginRejectsBlockedAccount(t *testing.T) {
	t.Parallel()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}
	repository := &fakeRepository{getByNickname: func(context.Context, string) (usermodel.User, error) {
		return usermodel.User{PasswordHash: string(passwordHash), IsBlocked: true}, nil
	}}

	_, err = New(repository, &fakeTokenIssuer{}).Login(context.Background(), LoginInput{
		Nickname: "Samir",
		Password: "password123",
	})
	if !errors.Is(err, ErrAccountBlocked) {
		t.Fatalf("Login() error = %v, want ErrAccountBlocked", err)
	}
}

func TestCurrentUserReturnsUnauthorizedWhenUserIsMissing(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{
		getByID: func(context.Context, uuid.UUID) (usermodel.User, error) {
			return usermodel.User{}, userrepository.ErrNotFound
		},
	}

	_, err := New(repository, &fakeTokenIssuer{}).CurrentUser(context.Background(), uuid.New())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("CurrentUser() error = %v, want ErrUnauthorized", err)
	}
}
