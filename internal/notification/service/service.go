package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	notificationmodel "github.com/sweetlife999/chain-of-trades-avito/internal/notification/model"
)

const maxPageLimit int32 = 100

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("notification not found")
)

type Repository interface {
	List(context.Context, uuid.UUID, notificationmodel.Filter) ([]notificationmodel.Notification, error)
	CountUnread(context.Context, uuid.UUID) (int64, error)
	MarkRead(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	MarkAllRead(context.Context, uuid.UUID) (int64, error)
}

type Service struct {
	repository Repository
}

func New(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(
	ctx context.Context,
	userID uuid.UUID,
	filter notificationmodel.Filter,
) (notificationmodel.Page, error) {
	if userID == uuid.Nil {
		return notificationmodel.Page{}, fmt.Errorf("%w: user id is required", ErrValidation)
	}
	if filter.Limit < 1 || filter.Limit > maxPageLimit {
		return notificationmodel.Page{}, fmt.Errorf(
			"%w: limit must be between 1 and %d",
			ErrValidation,
			maxPageLimit,
		)
	}
	if filter.Offset < 0 {
		return notificationmodel.Page{}, fmt.Errorf("%w: offset must not be negative", ErrValidation)
	}

	notifications, err := s.repository.List(ctx, userID, filter)
	if err != nil {
		return notificationmodel.Page{}, fmt.Errorf("list notifications: %w", err)
	}
	unreadCount, err := s.repository.CountUnread(ctx, userID)
	if err != nil {
		return notificationmodel.Page{}, fmt.Errorf("count unread notifications: %w", err)
	}

	return notificationmodel.Page{
		Notifications: notifications,
		UnreadCount:   unreadCount,
		Limit:         filter.Limit,
		Offset:        filter.Offset,
	}, nil
}

func (s *Service) MarkRead(
	ctx context.Context,
	userID uuid.UUID,
	notificationID uuid.UUID,
) error {
	if userID == uuid.Nil || notificationID == uuid.Nil {
		return fmt.Errorf("%w: ids are required", ErrValidation)
	}

	found, err := s.repository.MarkRead(ctx, userID, notificationID)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	if userID == uuid.Nil {
		return 0, fmt.Errorf("%w: user id is required", ErrValidation)
	}

	affected, err := s.repository.MarkAllRead(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	return affected, nil
}
