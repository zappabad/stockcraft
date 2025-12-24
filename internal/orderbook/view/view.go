package view

import (
	"sort"
	"sync"

	"github.com/zappabad/stockcraft/internal/orderbook/core"
)

// RestingOrder represents a snapshot of a resting order.
type RestingOrder struct {
	ID     core.OrderID
	UserID core.UserID
	Side   core.Side
	Price  core.PriceTicks
	Size   core.Size
	Time   int64
}

// Level represents aggregate size at a price level.
type Level struct {
	Price core.PriceTicks
	Size  core.Size
}

type orderState struct {
	userID core.UserID
	side   core.Side
	price  core.PriceTicks
	size   core.Size
	time   int64
}

// BookView maintains a read-only view of the orderbook state.
// It is thread-safe and returns copies (not internal references).
type BookView struct {
	mu     sync.RWMutex
	orders map[core.OrderID]orderState
	bids   map[core.PriceTicks]core.Size
	asks   map[core.PriceTicks]core.Size
	tape   *TradeTape
}

// NewBookView creates a new BookView with the given trade tape capacity.
func NewBookView(tapeCapacity int) *BookView {
	return &BookView{
		orders: map[core.OrderID]orderState{},
		bids:   map[core.PriceTicks]core.Size{},
		asks:   map[core.PriceTicks]core.Size{},
		tape:   NewTradeTape(tapeCapacity),
	}
}

// Apply processes an event and updates the view accordingly.
func (v *BookView) Apply(ev core.Event) {
	v.mu.Lock()
	defer v.mu.Unlock()

	switch e := ev.(type) {
	case core.TradeEvent:
		v.tape.Append(e)

	case core.OrderRestedEvent:
		v.orders[e.OrderID] = orderState{
			userID: e.UserID,
			side:   e.Side,
			price:  e.Price,
			size:   e.Size,
			time:   e.Time,
		}
		if e.Side == core.SideBuy {
			v.bids[e.Price] += e.Size
		} else {
			v.asks[e.Price] += e.Size
		}

	case core.OrderReducedEvent:
		st, ok := v.orders[e.OrderID]
		if !ok {
			// if this happens, your event stream is incomplete or out of order
			return
		}
		// delta is negative; update totals by delta
		if st.side == core.SideBuy {
			v.bids[st.price] += e.Delta
			if v.bids[st.price] <= 0 {
				delete(v.bids, st.price)
			}
		} else {
			v.asks[st.price] += e.Delta
			if v.asks[st.price] <= 0 {
				delete(v.asks, st.price)
			}
		}
		st.size = e.Remaining
		v.orders[e.OrderID] = st

	case core.OrderRemovedEvent:
		st, ok := v.orders[e.OrderID]
		if ok {
			if st.side == core.SideBuy {
				v.bids[st.price] -= st.size
				if v.bids[st.price] <= 0 {
					delete(v.bids, st.price)
				}
			} else {
				v.asks[st.price] -= st.size
				if v.asks[st.price] <= 0 {
					delete(v.asks, st.price)
				}
			}
			delete(v.orders, e.OrderID)
		}
	}
}

// Levels returns aggregate size at each price level, sorted best->worst.
// Returns a copy (not internal references).
func (v *BookView) Levels(side core.Side) []Level {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var src map[core.PriceTicks]core.Size
	if side == core.SideBuy {
		src = v.bids
	} else {
		src = v.asks
	}

	out := make([]Level, 0, len(src))
	for p, s := range src {
		out = append(out, Level{Price: p, Size: s})
	}

	sort.Slice(out, func(i, j int) bool {
		if side == core.SideBuy {
			return out[i].Price > out[j].Price // best bid is highest
		}
		return out[i].Price < out[j].Price // best ask is lowest
	})
	return out
}

// Orders returns all resting orders on a side, sorted by price (best first), then time, then id.
// Returns a copy (not internal references).
func (v *BookView) Orders(side core.Side) []RestingOrder {
	v.mu.RLock()
	defer v.mu.RUnlock()

	out := make([]RestingOrder, 0, len(v.orders))
	for id, st := range v.orders {
		if st.side != side {
			continue
		}
		out = append(out, RestingOrder{
			ID:     id,
			UserID: st.userID,
			Side:   st.side,
			Price:  st.price,
			Size:   st.size,
			Time:   st.time,
		})
	}

	// deterministic ordering for callers: best price then time then id
	sort.Slice(out, func(i, j int) bool {
		if out[i].Price != out[j].Price {
			if side == core.SideBuy {
				return out[i].Price > out[j].Price
			}
			return out[i].Price < out[j].Price
		}
		if out[i].Time != out[j].Time {
			return out[i].Time < out[j].Time
		}
		return out[i].ID < out[j].ID
	})

	return out
}

// TradesLast returns the last n trades in chronological order.
// Returns a copy (not internal references).
func (v *BookView) TradesLast(n int) []core.TradeEvent {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.tape.Last(n)
}
