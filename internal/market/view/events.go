package view

import (
	"github.com/zappabad/stockcraft/internal/market"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
)

// MarketEvent wraps a core event with its associated ticker.
type MarketEvent struct {
	Ticker market.TickerID
	Event  core.Event
}
