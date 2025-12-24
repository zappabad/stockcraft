package trader

import (
	"github.com/zappabad/stockcraft/internal/market"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
)

// TraderID uniquely identifies a trader.
type TraderID int64

// OrderIntent represents a trader's intention to place an order.
type OrderIntent struct {
	TickerID market.TickerID
	Kind     core.OrderKind
	Side     core.Side
	Price    core.PriceTicks // for limit orders only
	Size     core.Size
}

// TraderEventType indicates the type of trader event.
type TraderEventType int

const (
	TraderEventPlacedOrder TraderEventType = iota
	TraderEventRequestedApproval
	TraderEventCanceled
	TraderEventError
)

// TraderEvent represents an action or event from a trader.
type TraderEvent struct {
	TraderID TraderID
	Time     int64
	Type     TraderEventType
	Intent   *OrderIntent // optional, for PlacedOrder
	Message  string       // optional, for errors or info
}
