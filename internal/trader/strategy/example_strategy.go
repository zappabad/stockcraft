package strategy

import (
	"context"

	"github.com/zappabad/stockcraft/internal/orderbook/core"
	"github.com/zappabad/stockcraft/internal/trader"
)

// ExampleStrategy is a trivial strategy that places small bids when a spread exists.
type ExampleStrategy struct {
	traderID trader.TraderID
}

// NewExampleStrategy creates a new ExampleStrategy.
func NewExampleStrategy(traderID trader.TraderID) *ExampleStrategy {
	return &ExampleStrategy{traderID: traderID}
}

// Step implements Strategy.
func (s *ExampleStrategy) Step(ctx context.Context, now int64, mr MarketReader, nr NewsReader) ([]trader.OrderIntent, []trader.TraderEvent) {
	var intents []trader.OrderIntent
	var events []trader.TraderEvent

	tickers := mr.GetTickers()
	if len(tickers) == 0 {
		return nil, nil
	}

	// Pick the first ticker
	ticker := tickers[0]
	tid := ticker.TickerID()

	// Get best bid and ask
	bids, err := mr.GetLevels(tid, core.SideBuy)
	if err != nil {
		return nil, nil
	}
	asks, err := mr.GetLevels(tid, core.SideSell)
	if err != nil {
		return nil, nil
	}

	// If there's a spread, place a small bid below best ask
	if len(asks) > 0 {
		bestAsk := asks[0].Price
		bidPrice := bestAsk - 1
		if bidPrice > 0 {
			// Check if our bid would be best
			if len(bids) == 0 || bidPrice > bids[0].Price {
				intent := trader.OrderIntent{
					TickerID: tid,
					Kind:     core.OrderKindLimit,
					Side:     core.SideBuy,
					Price:    bidPrice,
					Size:     1,
				}
				intents = append(intents, intent)

				events = append(events, trader.TraderEvent{
					TraderID: s.traderID,
					Time:     now,
					Type:     trader.TraderEventPlacedOrder,
					Intent:   &intent,
				})
			}
		}
	}

	return intents, events
}
