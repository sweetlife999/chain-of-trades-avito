package repository

import (
	"context"
	"errors"
	"testing"

	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
)

type fakeQuerier struct {
	row db.GetAdminDashboardRow
	err error
}

func (f fakeQuerier) GetAdminDashboard(context.Context) (db.GetAdminDashboardRow, error) {
	return f.row, f.err
}

func TestGetMapsDashboardStatistics(t *testing.T) {
	t.Parallel()

	want := db.GetAdminDashboardRow{
		UsersTotal:         10,
		PickupPointsTotal:  2,
		ItemsTotal:         20,
		ItemsAvailable:     11,
		ItemsReserved:      3,
		ItemsTraded:        4,
		ItemsWithdrawn:     2,
		ExchangesTotal:     8,
		ExchangesProposed:  3,
		ExchangesConfirmed: 2,
		ExchangesCompleted: 2,
		ExchangesCancelled: 1,
	}

	dashboard, err := New(fakeQuerier{row: want}).Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if dashboard.UsersTotal != want.UsersTotal || dashboard.PickupPointsTotal != want.PickupPointsTotal {
		t.Fatalf("top-level statistics = %+v, want users=%d pickup_points=%d", dashboard, want.UsersTotal, want.PickupPointsTotal)
	}
	if dashboard.Items.Available != want.ItemsAvailable || dashboard.Items.Withdrawn != want.ItemsWithdrawn {
		t.Fatalf("item statistics = %+v", dashboard.Items)
	}
	if dashboard.Exchanges.Completed != want.ExchangesCompleted || dashboard.Exchanges.Cancelled != want.ExchangesCancelled {
		t.Fatalf("exchange statistics = %+v", dashboard.Exchanges)
	}
}

func TestGetWrapsQueryError(t *testing.T) {
	t.Parallel()

	want := errors.New("database unavailable")
	_, err := New(fakeQuerier{err: want}).Get(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Get() error = %v, want wrapped query error", err)
	}
}
