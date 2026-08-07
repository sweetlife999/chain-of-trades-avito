package service

import (
	"context"
	"errors"
	"testing"

	admindashboardmodel "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/model"
)

type fakeRepository struct {
	dashboard admindashboardmodel.Dashboard
	err       error
}

func (f fakeRepository) Get(context.Context) (admindashboardmodel.Dashboard, error) {
	return f.dashboard, f.err
}

func TestGetReturnsRepositoryDashboard(t *testing.T) {
	t.Parallel()

	want := admindashboardmodel.Dashboard{UsersTotal: 7, PickupPointsTotal: 3}
	got, err := New(fakeRepository{dashboard: want}).Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != want {
		t.Fatalf("Get() = %+v, want %+v", got, want)
	}
}

func TestGetWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	want := errors.New("database unavailable")
	_, err := New(fakeRepository{err: want}).Get(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Get() error = %v, want wrapped repository error", err)
	}
}
