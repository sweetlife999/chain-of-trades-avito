package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

const maxParticipants = 5

var ErrInvalidCycle = errors.New("invalid exchange cycle")

type Repository interface {
	FindNeighbors(context.Context, uuid.UUID) ([]exchangemodel.Node, error)
	SaveExchange(context.Context, exchangemodel.Exchange) (uuid.UUID, error)
}

type Service struct {
	repository Repository
}

type SearchResult struct {
	ExchangeID uuid.UUID
	Found      bool
}

func New(repository Repository) *Service {
	return &Service{repository: repository}
}

// FindCycle ищет первый обмен, который начинается со start и возвращается в него.
// Отсутствие подходящего обмена — нормальный результат: в этом случае возвращается nil, nil.
func (s *Service) FindCycle(ctx context.Context, start exchangemodel.Node) ([]exchangemodel.Node, error) {
	path := []exchangemodel.Node{start}
	visitedItems := map[uuid.UUID]struct{}{start.ItemID: {}}
	visitedOwners := map[uuid.UUID]struct{}{start.OwnerID: {}}

	var cycle []exchangemodel.Node

	var dfs func(exchangemodel.Node) (bool, error)
	dfs = func(current exchangemodel.Node) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		neighbors, err := s.repository.FindNeighbors(ctx, current.ItemID)
		if err != nil {
			return false, fmt.Errorf("find neighbors for item %s: %w", current.ItemID, err)
		}

		for _, next := range neighbors {
			// На глубине 5 ещё можно замкнуть путь, но нельзя добавить шестого участника.
			if next.ItemID == start.ItemID {
				if len(path) >= 2 {
					cycle = append([]exchangemodel.Node(nil), path...)
					return true, nil
				}

				continue
			}

			if len(path) >= maxParticipants {
				continue
			}

			if _, visited := visitedItems[next.ItemID]; visited {
				continue
			}

			if _, visited := visitedOwners[next.OwnerID]; visited {
				continue
			}

			visitedItems[next.ItemID] = struct{}{}
			visitedOwners[next.OwnerID] = struct{}{}
			path = append(path, next)

			found, err := dfs(next)

			path = path[:len(path)-1]
			delete(visitedItems, next.ItemID)
			delete(visitedOwners, next.OwnerID)

			if err != nil {
				return false, err
			}

			if found {
				return true, nil
			}
		}

		return false, nil
	}

	found, err := dfs(start)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, nil
	}

	return cycle, nil
}

// SaveCycle переводит найденный путь в участников обмена и сохраняет их одной транзакцией.
func (s *Service) SaveCycle(ctx context.Context, cycle []exchangemodel.Node) (uuid.UUID, error) {
	if err := validateCycle(cycle); err != nil {
		return uuid.Nil, err
	}

	participants := make([]exchangemodel.Participant, len(cycle))
	for index, node := range cycle {
		next := cycle[(index+1)%len(cycle)]
		participants[index] = exchangemodel.Participant{
			UserID:         node.OwnerID,
			GivesItemID:    node.ItemID,
			ReceivesItemID: next.ItemID,
			Position:       int32(index),
		}
	}

	id, err := s.repository.SaveExchange(ctx, exchangemodel.Exchange{Participants: participants})
	if err != nil {
		return uuid.Nil, fmt.Errorf("save cycle: %w", err)
	}

	return id, nil
}

// FindAndSave запускает полный сценарий: ищет обмен от нового объявления и,
// если находит, сохраняет его. Отсутствие обмена не считается ошибкой.
func (s *Service) FindAndSave(
	ctx context.Context,
	start exchangemodel.Node,
) (SearchResult, error) {
	cycle, err := s.FindCycle(ctx, start)
	if err != nil {
		return SearchResult{}, fmt.Errorf("search exchange: %w", err)
	}

	if cycle == nil {
		return SearchResult{}, nil
	}

	exchangeID, err := s.SaveCycle(ctx, cycle)
	if err != nil {
		return SearchResult{}, fmt.Errorf("persist found exchange: %w", err)
	}

	return SearchResult{
		ExchangeID: exchangeID,
		Found:      true,
	}, nil
}

func validateCycle(cycle []exchangemodel.Node) error {
	if len(cycle) < 2 || len(cycle) > maxParticipants {
		return fmt.Errorf("%w: participant count must be between 2 and %d", ErrInvalidCycle, maxParticipants)
	}

	items := make(map[uuid.UUID]struct{}, len(cycle))
	owners := make(map[uuid.UUID]struct{}, len(cycle))

	for _, node := range cycle {
		if node.ItemID == uuid.Nil || node.OwnerID == uuid.Nil {
			return fmt.Errorf("%w: item and owner IDs must not be empty", ErrInvalidCycle)
		}

		if _, exists := items[node.ItemID]; exists {
			return fmt.Errorf("%w: item %s is repeated", ErrInvalidCycle, node.ItemID)
		}
		items[node.ItemID] = struct{}{}

		if _, exists := owners[node.OwnerID]; exists {
			return fmt.Errorf("%w: owner %s is repeated", ErrInvalidCycle, node.OwnerID)
		}
		owners[node.OwnerID] = struct{}{}
	}

	return nil
}
