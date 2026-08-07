package model

type Dashboard struct {
	UsersTotal        int64
	PickupPointsTotal int64
	Items             ItemStatistics
	Exchanges         ExchangeStatistics
}

type ItemStatistics struct {
	Total     int64
	Available int64
	Reserved  int64
	Traded    int64
	Withdrawn int64
}

type ExchangeStatistics struct {
	Total     int64
	Proposed  int64
	Confirmed int64
	Completed int64
	Cancelled int64
}
