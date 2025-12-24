package core

import "strconv"

// Side represents the order side: buy or sell.
type Side uint8

const (
	SideBuy Side = iota
	SideSell
)

func (s Side) String() string {
	switch s {
	case SideBuy:
		return "BUY"
	case SideSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

// Opposite returns the opposite side.
func (s Side) Opposite() Side {
	if s == SideBuy {
		return SideSell
	}
	return SideBuy
}

// OrderKind represents the order type: limit or market.
type OrderKind uint8

const (
	OrderKindLimit OrderKind = iota
	OrderKindMarket
)

func (k OrderKind) String() string {
	switch k {
	case OrderKindLimit:
		return "LIMIT"
	case OrderKindMarket:
		return "MARKET"
	default:
		return "UNKNOWN"
	}
}

// PriceTicks represents price in integer ticks.
type PriceTicks int64

func (p PriceTicks) String() string { return strconv.FormatInt(int64(p), 10) }

// Size represents order quantity.
type Size int64

func (s Size) String() string { return strconv.FormatInt(int64(s), 10) }

// OrderID uniquely identifies an order.
type OrderID int64

// UserID identifies the user/trader who placed the order.
type UserID int64

// Order is an input/value object (safe to pass around).
// Core mutates its own internal resting orders, not this.
type Order struct {
	ID     OrderID
	UserID UserID
	Side   Side
	Kind   OrderKind
	Price  PriceTicks // limit only
	Size   Size       // requested size (for submits); remaining size (in reports)
	Time   int64      // unix nanos set by service layer
}

// IsFilled returns true if the order has no remaining size.
func (o Order) IsFilled() bool { return o.Size <= 0 }
