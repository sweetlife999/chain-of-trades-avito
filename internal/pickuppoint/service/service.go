package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	pickuppointmodel "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/model"
	pickuppointrepository "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/repository"
)

const (
	maxNameLength    = 120
	maxAddressLength = 500
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = pickuppointrepository.ErrNotFound
	ErrInUse      = pickuppointrepository.ErrInUse
)

type Repository interface {
	Create(context.Context, pickuppointmodel.NewPickupPoint) (pickuppointmodel.PickupPoint, error)
	GetByID(context.Context, uuid.UUID) (pickuppointmodel.PickupPoint, error)
	List(context.Context) ([]pickuppointmodel.PickupPoint, error)
	Update(context.Context, uuid.UUID, pickuppointmodel.Changes) (pickuppointmodel.PickupPoint, error)
	Delete(context.Context, uuid.UUID) error
}

type Service struct {
	repository Repository
}

type CreateInput struct {
	Name    string
	Address string
}

type UpdateInput struct {
	Name    *string
	Address *string
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
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (pickuppointmodel.PickupPoint, error) {
	name, err := cleanRequired("name", input.Name, maxNameLength)
	if err != nil {
		return pickuppointmodel.PickupPoint{}, err
	}
	address, err := cleanRequired("address", input.Address, maxAddressLength)
	if err != nil {
		return pickuppointmodel.PickupPoint{}, err
	}

	point, err := s.repository.Create(ctx, pickuppointmodel.NewPickupPoint{Name: name, Address: address})
	if err != nil {
		return pickuppointmodel.PickupPoint{}, fmt.Errorf("create pickup point: %w", err)
	}

	return point, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (pickuppointmodel.PickupPoint, error) {
	point, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return pickuppointmodel.PickupPoint{}, fmt.Errorf("get pickup point: %w", err)
	}

	return point, nil
}

func (s *Service) List(ctx context.Context) ([]pickuppointmodel.PickupPoint, error) {
	points, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pickup points: %w", err)
	}

	return points, nil
}

func (s *Service) Update(
	ctx context.Context,
	id uuid.UUID,
	input UpdateInput,
) (pickuppointmodel.PickupPoint, error) {
	if input.Name == nil && input.Address == nil {
		return pickuppointmodel.PickupPoint{}, validationError("at least one field is required")
	}

	changes := pickuppointmodel.Changes{}
	if input.Name != nil {
		name, err := cleanRequired("name", *input.Name, maxNameLength)
		if err != nil {
			return pickuppointmodel.PickupPoint{}, err
		}
		changes.Name = &name
	}
	if input.Address != nil {
		address, err := cleanRequired("address", *input.Address, maxAddressLength)
		if err != nil {
			return pickuppointmodel.PickupPoint{}, err
		}
		changes.Address = &address
	}

	point, err := s.repository.Update(ctx, id, changes)
	if err != nil {
		return pickuppointmodel.PickupPoint{}, fmt.Errorf("update pickup point: %w", err)
	}

	return point, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete pickup point: %w", err)
	}

	return nil
}

func cleanRequired(field, value string, maxLength int) (string, error) {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return "", validationError(field + " is required")
	}
	if utf8.RuneCountInString(cleaned) > maxLength {
		return "", validationError(fmt.Sprintf("%s must be at most %d characters", field, maxLength))
	}

	return cleaned, nil
}

func validationError(message string) error {
	return &ValidationError{message: message}
}
