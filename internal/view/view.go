package view

import (
	"sort"
	"sync"

	"github.com/zappabad/stockcraft/internal/engine"
)

type RestingOrder struct {
	ID     engine.OrderID
	UserID engine.UserID
	Side   engine.Side
	Price  engine.PriceTicks
	Size   engine.Size
	Time   int64
}

type Level struct {
	Price engine.PriceTicks
	Size  engine.Size
}

type orderState struct {
	userID engine.UserID
	side   engine.Side
	price  engine.PriceTicks
	size   engine.Size
	time   int64
}

type BookView struct {
	mu     sync.RWMutex
	orders map[engine.OrderID]orderState
	bids   map[engine.PriceTicks]engine.Size
	asks   map[engine.PriceTicks]engine.Size
	tape   *TradeTape
}

func NewBookView(tapeCapacity int) *BookView {
	return &BookView{
		orders: map[engine.OrderID]orderState{},
		bids:   map[engine.PriceTicks]engine.Size{},
		asks:   map[engine.PriceTicks]engine.Size{},
		tape:   NewTradeTape(tapeCapacity),
	}
}

func (v *BookView) Apply(ev engine.Event) {
	v.mu.Lock()
	defer v.mu.Unlock()

	switch e := ev.(type) {
	case engine.TradeEvent:
		v.tape.Append(e)

	case engine.OrderRestedEvent:
		v.orders[e.OrderID] = orderState{
			userID: e.UserID,
			side:   e.Side,
			price:  e.Price,
			size:   e.Size,
			time:   e.Time,
		}
		if e.Side == engine.SideBuy {
			v.bids[e.Price] += e.Size
		} else {
			v.asks[e.Price] += e.Size
		}

	case engine.OrderReducedEvent:
		st, ok := v.orders[e.OrderID]
		if !ok {
			// if this happens, your event stream is incomplete or out of order
			return
		}
		// delta is negative; update totals by delta
		if st.side == engine.SideBuy {
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

	case engine.OrderRemovedEvent:
		st, ok := v.orders[e.OrderID]
		if ok {
			if st.side == engine.SideBuy {
				v.bids[st.price] -= e.Remaining
				if v.bids[st.price] <= 0 {
					delete(v.bids, st.price)
				}
			} else {
				v.asks[st.price] -= e.Remaining
				if v.asks[st.price] <= 0 {
					delete(v.asks, st.price)
				}
			}
			delete(v.orders, e.OrderID)
		}
	}
}

func (v *BookView) Levels(side engine.Side) []Level {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var src map[engine.PriceTicks]engine.Size
	if side == engine.SideBuy {
		src = v.bids
	} else {
		src = v.asks
	}

	out := make([]Level, 0, len(src))
	for p, s := range src {
		out = append(out, Level{Price: p, Size: s})
	}

	sort.Slice(out, func(i, j int) bool {
		if side == engine.SideBuy {
			return out[i].Price > out[j].Price
		}
		return out[i].Price < out[j].Price
	})
	return out
}

func (v *BookView) Orders(side engine.Side) []RestingOrder {
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
			if side == engine.SideBuy {
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

func (v *BookView) TradesLast(n int) []engine.TradeEvent {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.tape.Last(n)
}
