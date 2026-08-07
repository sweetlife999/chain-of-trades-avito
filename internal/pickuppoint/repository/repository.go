package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	pickuppointmodel "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/model"
)

var (
	ErrNotFound = errors.New("pickup point not found")
	ErrInUse    = errors.New("pickup point is in use")
)

type Repository struct {
	queries *db.Queries
}

func New(queries *db.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, point pickuppointmodel.NewPickupPoint) (pickuppointmodel.PickupPoint, error) {
	created, err := r.queries.CreatePickupPoint(ctx, db.CreatePickupPointParams{
		Name:    point.Name,
		Address: point.Address,
	})
	if err != nil {
		return pickuppointmodel.PickupPoint{}, fmt.Errorf("create pickup point: %w", translateError(err))
	}

	return toModel(created), nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (pickuppointmodel.PickupPoint, error) {
	found, err := r.queries.GetPickupPointByID(ctx, pgUUID(id))
	if err != nil {
		return pickuppointmodel.PickupPoint{}, fmt.Errorf("get pickup point by id: %w", translateError(err))
	}

	return toModel(found), nil
}

func (r *Repository) List(ctx context.Context) ([]pickuppointmodel.PickupPoint, error) {
	rows, err := r.queries.ListPickupPoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pickup points: %w", err)
	}

	points := make([]pickuppointmodel.PickupPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, toModel(row))
	}

	return points, nil
}

func (r *Repository) Update(
	ctx context.Context,
	id uuid.UUID,
	changes pickuppointmodel.Changes,
) (pickuppointmodel.PickupPoint, error) {
	updated, err := r.queries.UpdatePickupPoint(ctx, db.UpdatePickupPointParams{
		Name:    optionalText(changes.Name),
		Address: optionalText(changes.Address),
		ID:      pgUUID(id),
	})
	if err != nil {
		return pickuppointmodel.PickupPoint{}, fmt.Errorf("update pickup point: %w", translateError(err))
	}

	return toModel(updated), nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := r.queries.DeletePickupPoint(ctx, pgUUID(id))
	if err != nil {
		return fmt.Errorf("delete pickup point: %w", translateError(err))
	}
	if deleted == 0 {
		return ErrNotFound
	}

	return nil
}

func optionalText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}

func toModel(point db.PickupPoint) pickuppointmodel.PickupPoint {
	return pickuppointmodel.PickupPoint{
		ID:        uuid.UUID(point.ID.Bytes),
		Name:      point.Name,
		Address:   point.Address,
		CreatedAt: point.CreatedAt.Time,
		UpdatedAt: point.UpdatedAt.Time,
	}
}

func translateError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrInUse
	}

	return err
}
