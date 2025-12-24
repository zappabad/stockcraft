package engine

type Event interface {
	isEvent()
}

type RemoveReason uint8

const (
	RemoveReasonFilled RemoveReason = iota
	RemoveReasonCanceled
)

type OrderRestedEvent struct {
	OrderID OrderID
	UserID  UserID
	Side    Side
	Price   PriceTicks
	Size    Size
	Time    int64
}

func (OrderRestedEvent) isEvent() {}

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
