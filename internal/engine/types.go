package engine

import "strconv"

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

type PriceTicks int64
type Size int64

func (p PriceTicks) String() string { return strconv.FormatInt(int64(p), 10) }
func (s Size) String() string       { return strconv.FormatInt(int64(s), 10) }

type OrderID int64
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
	Time   int64      // unix nanos set by API layer
}

func (o Order) IsFilled() bool { return o.Size <= 0 }
