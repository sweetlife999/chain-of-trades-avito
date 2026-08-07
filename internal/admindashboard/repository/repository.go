package repository

import (
	"context"
	"fmt"

	admindashboardmodel "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/model"
	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

type Querier interface {
	GetAdminDashboard(context.Context) (db.GetAdminDashboardRow, error)
}

type Repository struct {
	queries Querier
}

func New(queries Querier) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Get(ctx context.Context) (admindashboardmodel.Dashboard, error) {
	row, err := r.queries.GetAdminDashboard(ctx)
	if err != nil {
		return admindashboardmodel.Dashboard{}, fmt.Errorf("get admin dashboard: %w", err)
	}

	return toModel(row), nil
}

func toModel(row db.GetAdminDashboardRow) admindashboardmodel.Dashboard {
	return admindashboardmodel.Dashboard{
		UsersTotal:        row.UsersTotal,
		PickupPointsTotal: row.PickupPointsTotal,
		Items: admindashboardmodel.ItemStatistics{
			Total:     row.ItemsTotal,
			Available: row.ItemsAvailable,
			Reserved:  row.ItemsReserved,
			Traded:    row.ItemsTraded,
			Withdrawn: row.ItemsWithdrawn,
		},
		Exchanges: admindashboardmodel.ExchangeStatistics{
			Total:     row.ExchangesTotal,
			Proposed:  row.ExchangesProposed,
			Confirmed: row.ExchangesConfirmed,
			Completed: row.ExchangesCompleted,
			Cancelled: row.ExchangesCancelled,
		},
	}
}
