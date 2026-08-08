package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	adminexchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/model"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
)

const (
	DefaultLimit int32 = 20
	MaxLimit     int32 = 100
)

var (
	ErrNotFound   = userrepository.ErrNotFound
	ErrValidation = errors.New("validation error")
)

type UserRepository interface {
	GetByID(context.Context, uuid.UUID) (usermodel.User, error)
}

type ExchangeRepository interface {
	ListActiveByUser(context.Context, uuid.UUID, int32, int32) ([]exchangemodel.Details, error)
	CountActiveByUser(context.Context, uuid.UUID) (int64, error)
}

type Service struct {
	users     UserRepository
	exchanges ExchangeRepository
}

func New(users UserRepository, exchanges ExchangeRepository) *Service {
	return &Service{users: users, exchanges: exchanges}
}

func (s *Service) ListActiveByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit int32,
	offset int32,
) (adminexchangemodel.Page, error) {
	if limit < 1 || limit > MaxLimit || offset < 0 {
		return adminexchangemodel.Page{}, ErrValidation
	}

	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return adminexchangemodel.Page{}, fmt.Errorf("get user: %w", err)
	}

	exchanges, err := s.exchanges.ListActiveByUser(ctx, userID, limit, offset)
	if err != nil {
		return adminexchangemodel.Page{}, fmt.Errorf("list active exchanges: %w", err)
	}
	if exchanges == nil {
		exchanges = []exchangemodel.Details{}
	}

	total, err := s.exchanges.CountActiveByUser(ctx, userID)
	if err != nil {
		return adminexchangemodel.Page{}, fmt.Errorf("count active exchanges: %w", err)
	}

	return adminexchangemodel.Page{
		Exchanges: exchanges,
		Limit:     limit,
		Offset:    offset,
		Total:     total,
	}, nil
}
