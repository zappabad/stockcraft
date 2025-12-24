package view

import (
	"sync"

	"github.com/zappabad/stockcraft/internal/market"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
	orderbookservice "github.com/zappabad/stockcraft/internal/orderbook/service"
)

// BestPrices holds the current best bid/ask and last trade info for a ticker.
type BestPrices struct {
	BidPrice  core.PriceTicks
	BidSize   core.Size
	BidOK     bool
	AskPrice  core.PriceTicks
	AskSize   core.Size
	AskOK     bool
	LastPrice core.PriceTicks
	LastTime  int64
	HasLast   bool
}

// MarketSnapshot is a point-in-time snapshot of all tickers.
type MarketSnapshot struct {
	ByTicker map[market.TickerID]BestPrices
}

// MarketView maintains the aggregate market state across all tickers.
type MarketView struct {
	mu        sync.RWMutex
	lastTrade map[market.TickerID]core.TradeEvent
}

// NewMarketView creates a new MarketView.
func NewMarketView() *MarketView {
	return &MarketView{
		lastTrade: make(map[market.TickerID]core.TradeEvent),
	}
}

// Apply updates the view with an event from a specific ticker's orderbook.
func (v *MarketView) Apply(tid market.TickerID, ev core.Event, book *orderbookservice.Service) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Update last trade from TradeEvent
	if trade, ok := ev.(core.TradeEvent); ok {
		v.lastTrade[tid] = trade
	}
}

// Snapshot returns a deep copy of the current market state.
// For best bid/ask, it queries each book's levels (acceptable for TUI scale).
func (v *MarketView) Snapshot() MarketSnapshot {
	v.mu.RLock()
	defer v.mu.RUnlock()

	snap := MarketSnapshot{
		ByTicker: make(map[market.TickerID]BestPrices, len(v.lastTrade)),
	}

	// Copy last trades
	for tid, trade := range v.lastTrade {
		bp := BestPrices{
			LastPrice: trade.Price,
			LastTime:  trade.Time,
			HasLast:   true,
		}
		snap.ByTicker[tid] = bp
	}

	return snap
}

// SnapshotWithBooks returns a snapshot including best bid/ask from the orderbooks.
func (v *MarketView) SnapshotWithBooks(books map[market.TickerID]*orderbookservice.Service) MarketSnapshot {
	v.mu.RLock()
	defer v.mu.RUnlock()

	snap := MarketSnapshot{
		ByTicker: make(map[market.TickerID]BestPrices, len(books)),
	}

	for tid, book := range books {
		var bp BestPrices

		// Get best bid
		bids := book.GetLevels(core.SideBuy)
		if len(bids) > 0 {
			bp.BidPrice = bids[0].Price
			bp.BidSize = bids[0].Size
			bp.BidOK = true
		}

		// Get best ask
		asks := book.GetLevels(core.SideSell)
		if len(asks) > 0 {
			bp.AskPrice = asks[0].Price
			bp.AskSize = asks[0].Size
			bp.AskOK = true
		}

		// Get last trade
		if trade, ok := v.lastTrade[tid]; ok {
			bp.LastPrice = trade.Price
			bp.LastTime = trade.Time
			bp.HasLast = true
		}

		snap.ByTicker[tid] = bp
	}

	return snap
}
