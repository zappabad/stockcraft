package core

// Event is the interface for all orderbook events.
type Event interface {
	isEvent()
}

// RemoveReason indicates why an order was removed from the book.
type RemoveReason uint8

const (
	RemoveReasonFilled RemoveReason = iota
	RemoveReasonCanceled
)

func (r RemoveReason) String() string {
	switch r {
	case RemoveReasonFilled:
		return "FILLED"
	case RemoveReasonCanceled:
		return "CANCELED"
	default:
		return "UNKNOWN"
	}
}

// TradeEvent is emitted when a trade occurs.
type TradeEvent struct {
	Price     PriceTicks
	Size      Size
	TakerSide Side
	Time      int64

	TakerOrderID OrderID
	TakerUserID  UserID
	MakerOrderID OrderID
	MakerUserID  UserID
}

func (TradeEvent) isEvent() {}

// OrderRestedEvent is emitted when an order rests on the book.
type OrderRestedEvent struct {
	OrderID OrderID
	UserID  UserID
	Side    Side
	Price   PriceTicks
	Size    Size
	Time    int64
}

func (OrderRestedEvent) isEvent() {}

// OrderReducedEvent is emitted when a resting order is partially filled.
type OrderReducedEvent struct {
	OrderID   OrderID
	Delta     Size // negative number (e.g. -5)
	Remaining Size
	Price     PriceTicks
	Side      Side
	UserID    UserID
	MatchTime int64
}

func (OrderReducedEvent) isEvent() {}

// OrderRemovedEvent is emitted when an order is fully removed from the book.
type OrderRemovedEvent struct {
	OrderID   OrderID
	Reason    RemoveReason
	Remaining Size // size that was removed at time of removal (usually 0 for filled; >0 for cancel)
	Price     PriceTicks
	Side      Side
	UserID    UserID
	Time      int64
}

func (OrderRemovedEvent) isEvent() {}
