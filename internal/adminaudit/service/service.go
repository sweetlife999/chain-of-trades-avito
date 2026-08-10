package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	adminauditmodel "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/model"
	adminauditrepository "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/repository"
)

const (
	DefaultLimit int32 = 20
	MaxLimit     int32 = 100
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = adminauditrepository.ErrNotFound
	ErrConflict   = adminauditrepository.ErrConflict
)

type Repository interface {
	BlockUser(context.Context, uuid.UUID, uuid.UUID) (adminauditmodel.UserBlockState, error)
	UnblockUser(context.Context, uuid.UUID, uuid.UUID) (adminauditmodel.UserBlockState, error)
	List(context.Context, adminauditmodel.Filter) ([]adminauditmodel.Entry, error)
	Count(context.Context, adminauditmodel.Filter) (int64, error)
}

type Service struct{ repository Repository }
type ValidationError struct{ message string }

func (e *ValidationError) Error() string { return e.message }
func (e *ValidationError) Unwrap() error { return ErrValidation }
func New(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) BlockUser(ctx context.Context, adminID, userID uuid.UUID) (adminauditmodel.UserBlockState, error) {
	if adminID == uuid.Nil || userID == uuid.Nil {
		return adminauditmodel.UserBlockState{}, validationError("admin and user ids are required")
	}
	if adminID == userID {
		return adminauditmodel.UserBlockState{}, validationError("administrator cannot block themselves")
	}
	state, err := s.repository.BlockUser(ctx, adminID, userID)
	if err != nil {
		return adminauditmodel.UserBlockState{}, fmt.Errorf("block user: %w", err)
	}
	return state, nil
}

func (s *Service) UnblockUser(ctx context.Context, adminID, userID uuid.UUID) (adminauditmodel.UserBlockState, error) {
	if adminID == uuid.Nil || userID == uuid.Nil {
		return adminauditmodel.UserBlockState{}, validationError("admin and user ids are required")
	}
	state, err := s.repository.UnblockUser(ctx, adminID, userID)
	if err != nil {
		return adminauditmodel.UserBlockState{}, fmt.Errorf("unblock user: %w", err)
	}
	return state, nil
}

func (s *Service) List(ctx context.Context, filter adminauditmodel.Filter) (adminauditmodel.Page, error) {
	filter.Action = strings.TrimSpace(filter.Action)
	if filter.Limit == 0 {
		filter.Limit = DefaultLimit
	}
	if filter.Limit < 1 || filter.Limit > MaxLimit {
		return adminauditmodel.Page{}, validationError("limit must be between 1 and 100")
	}
	if filter.Offset < 0 {
		return adminauditmodel.Page{}, validationError("offset must be non-negative")
	}
	if filter.Action != "" && !slices.Contains(adminauditmodel.Actions, filter.Action) {
		return adminauditmodel.Page{}, validationError("unknown admin action")
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return adminauditmodel.Page{}, validationError("from must be before or equal to to")
	}
	entries, err := s.repository.List(ctx, filter)
	if err != nil {
		return adminauditmodel.Page{}, fmt.Errorf("list audit entries: %w", err)
	}
	total, err := s.repository.Count(ctx, filter)
	if err != nil {
		return adminauditmodel.Page{}, fmt.Errorf("count audit entries: %w", err)
	}
	if entries == nil {
		entries = []adminauditmodel.Entry{}
	}
	return adminauditmodel.Page{Entries: entries, Limit: filter.Limit, Offset: filter.Offset, Total: total}, nil
}

func validationError(message string) error { return &ValidationError{message: message} }
