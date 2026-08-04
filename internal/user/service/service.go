package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
)

const (
	minNicknameLength = 3
	maxNicknameLength = 32
	minPasswordLength = 8
	maxPasswordLength = 72
)

var (
	ErrValidation    = errors.New("validation error")
	ErrNotFound      = userrepository.ErrNotFound
	ErrNicknameTaken = userrepository.ErrNicknameTaken
)

type Repository interface {
	Create(context.Context, usermodel.NewUser) (usermodel.User, error)
	GetByID(context.Context, uuid.UUID) (usermodel.User, error)
	Update(context.Context, uuid.UUID, usermodel.Changes) (usermodel.User, error)
}

type Service struct {
	repository Repository
	bcryptCost int
}

type CreateInput struct {
	Nickname    string
	Password    string
	PhotoURL    *string
	Description string
}

type UpdateInput struct {
	Nickname    *string
	PhotoURL    *string
	Description *string
}

type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

func New(repository Repository) *Service {
	return newWithCost(repository, bcrypt.DefaultCost)
}

func newWithCost(repository Repository, bcryptCost int) *Service {
	return &Service{repository: repository, bcryptCost: bcryptCost}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (usermodel.User, error) {
	nickname := strings.TrimSpace(input.Nickname)
	if err := validateNickname(nickname); err != nil {
		return usermodel.User{}, err
	}

	if len(input.Password) < minPasswordLength || len(input.Password) > maxPasswordLength {
		return usermodel.User{}, validationError("password must contain from 8 to 72 bytes")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), s.bcryptCost)
	if err != nil {
		return usermodel.User{}, fmt.Errorf("hash password: %w", err)
	}

	return s.repository.Create(ctx, usermodel.NewUser{
		Nickname:     nickname,
		PasswordHash: string(passwordHash),
		PhotoURL:     cleanOptional(input.PhotoURL),
		Description:  strings.TrimSpace(input.Description),
	})
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (usermodel.User, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (usermodel.User, error) {
	if input.Nickname == nil && input.PhotoURL == nil && input.Description == nil {
		return usermodel.User{}, validationError("at least one field must be provided")
	}

	changes := usermodel.Changes{
		PhotoURL:    cleanOptional(input.PhotoURL),
		Description: cleanOptional(input.Description),
	}

	if input.Nickname != nil {
		nickname := strings.TrimSpace(*input.Nickname)
		if err := validateNickname(nickname); err != nil {
			return usermodel.User{}, err
		}
		changes.Nickname = &nickname
	}

	return s.repository.Update(ctx, id, changes)
}

func validateNickname(nickname string) error {
	length := utf8.RuneCountInString(nickname)
	if length < minNicknameLength || length > maxNicknameLength {
		return validationError("nickname must contain from 3 to 32 characters")
	}

	return nil
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}

	cleaned := strings.TrimSpace(*value)
	return &cleaned
}

func validationError(message string) error {
	return &ValidationError{message: message}
}
