package view

import "github.com/zappabad/stockcraft/internal/orderbook/core"

// TradeTape is a ring buffer for storing trade events (bounded memory).
type TradeTape struct {
	buf   []core.TradeEvent
	size  int
	start int
	count int
}

// NewTradeTape creates a new TradeTape with the given capacity.
func NewTradeTape(capacity int) *TradeTape {
	if capacity <= 0 {
		capacity = 1
	}
	return &TradeTape{
		buf:  make([]core.TradeEvent, capacity),
		size: capacity,
	}
}

// Append adds a trade event to the tape.
func (t *TradeTape) Append(tr core.TradeEvent) {
	if t.count < t.size {
		t.buf[(t.start+t.count)%t.size] = tr
		t.count++
		return
	}
	// overwrite oldest
	t.buf[t.start] = tr
	t.start = (t.start + 1) % t.size
}

// Last returns the last n trade events in chronological order.
// Returns a copy (not internal references).
func (t *TradeTape) Last(n int) []core.TradeEvent {
	if n <= 0 || t.count == 0 {
		return nil
	}
	if n > t.count {
		n = t.count
	}
	out := make([]core.TradeEvent, n)
	// take last n in chronological order
	first := (t.start + (t.count - n)) % t.size
	for i := 0; i < n; i++ {
		out[i] = t.buf[(first+i)%t.size]
	}
	return out
}

// Count returns the number of trades in the tape.
func (t *TradeTape) Count() int {
	return t.count
}
