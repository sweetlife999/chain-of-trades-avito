package dto

import (
	"testing"

	admindashboardmodel "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/model"
)

func TestFromModel(t *testing.T) {
	t.Parallel()

	response := FromModel(admindashboardmodel.Dashboard{
		UsersTotal:        5,
		PickupPointsTotal: 2,
		Items: admindashboardmodel.ItemStatistics{
			Total:     8,
			Available: 3,
		},
		Exchanges: admindashboardmodel.ExchangeStatistics{
			Total:     4,
			Completed: 1,
		},
	})

	if response.UsersTotal != 5 || response.PickupPointsTotal != 2 {
		t.Fatalf("top-level response = %+v", response)
	}
	if response.Items.Total != 8 || response.Items.Available != 3 {
		t.Fatalf("items response = %+v", response.Items)
	}
	if response.Exchanges.Total != 4 || response.Exchanges.Completed != 1 {
		t.Fatalf("exchanges response = %+v", response.Exchanges)
	}
}
