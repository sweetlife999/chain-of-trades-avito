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
	IsAdmin(context.Context, uuid.UUID) (bool, error)
	CanAuthenticate(context.Context, uuid.UUID) (bool, error)
	Update(context.Context, uuid.UUID, usermodel.Changes) (usermodel.User, error)
	Block(context.Context, uuid.UUID, uuid.UUID) error
	ListBlocked(context.Context, uuid.UUID) ([]usermodel.BlockedUser, error)
	Unblock(context.Context, uuid.UUID, uuid.UUID) error
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

func (s *Service) IsAdmin(ctx context.Context, id uuid.UUID) (bool, error) {
	isAdmin, err := s.repository.IsAdmin(ctx, id)
	if err != nil {
		return false, fmt.Errorf("check admin access: %w", err)
	}

	return isAdmin, nil
}

func (s *Service) CanAuthenticate(ctx context.Context, id uuid.UUID) (bool, error) {
	allowed, err := s.repository.CanAuthenticate(ctx, id)
	if err != nil {
		return false, fmt.Errorf("check authentication access: %w", err)
	}

	return allowed, nil
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

func (s *Service) Block(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	if blockerID == blockedID {
		return validationError("user cannot block themselves")
	}

	if err := s.repository.Block(ctx, blockerID, blockedID); err != nil {
		return fmt.Errorf("block user: %w", err)
	}

	return nil
}

func (s *Service) ListBlocked(ctx context.Context, blockerID uuid.UUID) ([]usermodel.BlockedUser, error) {
	blocked, err := s.repository.ListBlocked(ctx, blockerID)
	if err != nil {
		return nil, fmt.Errorf("list blocked users: %w", err)
	}
	if blocked == nil {
		return []usermodel.BlockedUser{}, nil
	}

	return blocked, nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	if blockerID == blockedID {
		return validationError("user cannot unblock themselves")
	}

	if err := s.repository.Unblock(ctx, blockerID, blockedID); err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}

	return nil
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
