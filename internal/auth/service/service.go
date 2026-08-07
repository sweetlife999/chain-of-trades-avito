package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid nickname or password")
	ErrUnauthorized       = errors.New("unauthorized")
)

type Repository interface {
	GetByNickname(context.Context, string) (usermodel.User, error)
	GetByID(context.Context, uuid.UUID) (usermodel.User, error)
}

type TokenIssuer interface {
	Generate(uuid.UUID) (string, time.Time, error)
}

type Service struct {
	repository Repository
	tokens     TokenIssuer
}

type LoginInput struct {
	Nickname string
	Password string
}

type LoginResult struct {
	User      usermodel.User
	Token     string
	ExpiresAt time.Time
}

func New(repository Repository, tokens TokenIssuer) *Service {
	return &Service{
		repository: repository,
		tokens:     tokens,
	}
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	user, err := s.repository.GetByNickname(ctx, strings.TrimSpace(input.Nickname))
	if err != nil {
		if errors.Is(err, userrepository.ErrNotFound) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("find user for login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	signedToken, expiresAt, err := s.tokens.Generate(user.ID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate access token: %w", err)
	}

	return LoginResult{
		User:      user,
		Token:     signedToken,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) CurrentUser(ctx context.Context, userID uuid.UUID) (usermodel.User, error) {
	user, err := s.repository.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userrepository.ErrNotFound) {
			return usermodel.User{}, ErrUnauthorized
		}
		return usermodel.User{}, fmt.Errorf("get current user: %w", err)
	}

	return user, nil
}
