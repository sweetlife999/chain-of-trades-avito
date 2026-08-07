package dto

import admindashboardmodel "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/model"

type DashboardResponse struct {
	UsersTotal        int64                      `json:"users_total"`
	PickupPointsTotal int64                      `json:"pickup_points_total"`
	Items             ItemStatisticsResponse     `json:"items"`
	Exchanges         ExchangeStatisticsResponse `json:"exchanges"`
}

type ItemStatisticsResponse struct {
	Total     int64 `json:"total"`
	Available int64 `json:"available"`
	Reserved  int64 `json:"reserved"`
	Traded    int64 `json:"traded"`
	Withdrawn int64 `json:"withdrawn"`
}

type ExchangeStatisticsResponse struct {
	Total     int64 `json:"total"`
	Proposed  int64 `json:"proposed"`
	Confirmed int64 `json:"confirmed"`
	Completed int64 `json:"completed"`
	Cancelled int64 `json:"cancelled"`
}

type DashboardError struct {
	Error string `json:"error"`
}

func FromModel(dashboard admindashboardmodel.Dashboard) DashboardResponse {
	return DashboardResponse{
		UsersTotal:        dashboard.UsersTotal,
		PickupPointsTotal: dashboard.PickupPointsTotal,
		Items: ItemStatisticsResponse{
			Total:     dashboard.Items.Total,
			Available: dashboard.Items.Available,
			Reserved:  dashboard.Items.Reserved,
			Traded:    dashboard.Items.Traded,
			Withdrawn: dashboard.Items.Withdrawn,
		},
		Exchanges: ExchangeStatisticsResponse{
			Total:     dashboard.Exchanges.Total,
			Proposed:  dashboard.Exchanges.Proposed,
			Confirmed: dashboard.Exchanges.Confirmed,
			Completed: dashboard.Exchanges.Completed,
			Cancelled: dashboard.Exchanges.Cancelled,
		},
	}
}
