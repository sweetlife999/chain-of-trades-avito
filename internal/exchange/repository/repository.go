package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

type neighborQueries interface {
	FindExchangeNeighbors(context.Context, pgtype.UUID) ([]db.FindExchangeNeighborsRow, error)
}

type Repository struct {
	queries neighborQueries
}

func New(queries neighborQueries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) FindNeighbors(ctx context.Context, itemID uuid.UUID) ([]exchangemodel.Node, error) {
	rows, err := r.queries.FindExchangeNeighbors(ctx, pgUUID(itemID))
	if err != nil {
		return nil, fmt.Errorf("find exchange neighbors: %w", err)
	}

	neighbors := make([]exchangemodel.Node, 0, len(rows))
	for _, row := range rows {
		neighbors = append(neighbors, exchangemodel.Node{
			ItemID:  uuid.UUID(row.ID.Bytes),
			OwnerID: uuid.UUID(row.OwnerID.Bytes),
		})
	}

	return neighbors, nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}
