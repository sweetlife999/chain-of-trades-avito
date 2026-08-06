package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

type neighborQueries interface {
	FindExchangeNeighbors(context.Context, pgtype.UUID) ([]db.FindExchangeNeighborsRow, error)
	HasUserBlockConflict(context.Context, db.HasUserBlockConflictParams) (bool, error)
}

type Repository struct {
	queries      neighborQueries
	reads        exchangeReadQueries
	transactions transactionManager
}

func New(pool *pgxpool.Pool) *Repository {
	queries := db.New(pool)
	return &Repository{
		queries:      queries,
		reads:        queries,
		transactions: &pgxTransactionManager{pool: pool},
	}
}

func newRepository(queries neighborQueries, transactions transactionManager) *Repository {
	return &Repository{
		queries:      queries,
		transactions: transactions,
	}
}

func newRepositoryWithReads(reads exchangeReadQueries) *Repository {
	return &Repository{reads: reads}
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

func (r *Repository) HasUserBlockConflict(
	ctx context.Context,
	candidateUserID uuid.UUID,
	pathUserIDs []uuid.UUID,
) (bool, error) {
	path := make([]pgtype.UUID, len(pathUserIDs))
	for index, id := range pathUserIDs {
		path[index] = pgUUID(id)
	}

	conflict, err := r.queries.HasUserBlockConflict(ctx, db.HasUserBlockConflictParams{
		CandidateUserID: pgUUID(candidateUserID),
		PathUserIds:     path,
	})
	if err != nil {
		return false, fmt.Errorf("check user block conflict: %w", err)
	}

	return conflict, nil
}

func (r *Repository) SaveExchange(
	ctx context.Context,
	exchange exchangemodel.Exchange,
) (uuid.UUID, error) {
	var exchangeID uuid.UUID

	err := r.transactions.WithinTransaction(ctx, func(queries exchangeWriteQueries) error {
		id, err := queries.CreateExchange(ctx)
		if err != nil {
			return fmt.Errorf("create exchange: %w", err)
		}

		exchangeID = uuid.UUID(id.Bytes)

		for _, participant := range exchange.Participants {
			err := queries.CreateExchangeParticipant(ctx, db.CreateExchangeParticipantParams{
				ChainID:        id,
				UserID:         pgUUID(participant.UserID),
				GivesItemID:    pgUUID(participant.GivesItemID),
				ReceivesItemID: pgUUID(participant.ReceivesItemID),
				Position:       participant.Position,
			})
			if err != nil {
				return fmt.Errorf("create exchange participant at position %d: %w", participant.Position, err)
			}
		}

		return nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("save exchange: %w", err)
	}

	return exchangeID, nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}
