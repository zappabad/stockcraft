package strategy

import (
	"context"

	"github.com/zappabad/stockcraft/internal/market"
	marketview "github.com/zappabad/stockcraft/internal/market/view"
	"github.com/zappabad/stockcraft/internal/news"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
	orderbookview "github.com/zappabad/stockcraft/internal/orderbook/view"
	"github.com/zappabad/stockcraft/internal/trader"
)

// MarketReader provides read-only access to market data.
type MarketReader interface {
	Snapshot() marketview.MarketSnapshot
	GetLevels(tid market.TickerID, side core.Side) ([]orderbookview.Level, error)
	GetTradesLast(tid market.TickerID, n int) ([]core.TradeEvent, error)
	GetTickers() []market.Ticker
}

// NewsReader provides read-only access to news data.
type NewsReader interface {
	Latest(n int) []news.NewsItem
}

// OrderSender provides the ability to send orders to the market.
type OrderSender interface {
	SubmitLimit(ctx context.Context, tid market.TickerID, userID core.UserID, side core.Side, price core.PriceTicks, size core.Size) (core.SubmitReport, error)
	SubmitMarket(ctx context.Context, tid market.TickerID, userID core.UserID, side core.Side, size core.Size) (core.SubmitReport, error)
	Cancel(ctx context.Context, tid market.TickerID, orderID core.OrderID) (core.CancelReport, error)
}

// Strategy is the interface for trading strategies.
type Strategy interface {
	// Step is called on each tick. Returns order intents and any events to publish.
	Step(ctx context.Context, now int64, mr MarketReader, nr NewsReader) ([]trader.OrderIntent, []trader.TraderEvent)
}
