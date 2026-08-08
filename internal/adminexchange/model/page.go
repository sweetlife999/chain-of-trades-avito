package model

import exchangemodel "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/model"

type Page struct {
	Exchanges []exchangemodel.Details
	Limit     int32
	Offset    int32
	Total     int64
}
