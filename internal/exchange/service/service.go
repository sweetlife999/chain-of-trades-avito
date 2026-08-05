package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"
)

const maxParticipants = 5

type Repository interface {
	FindNeighbors(context.Context, uuid.UUID) ([]exchangemodel.Node, error)
}

type Service struct {
	repository Repository
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
