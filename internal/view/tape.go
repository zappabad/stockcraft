package view

import "github.com/zappabad/stockcraft/internal/engine"

// Ring buffer trade tape (bounded memory).
type TradeTape struct {
	buf   []engine.TradeEvent
	size  int
	start int
	count int
}

func NewTradeTape(capacity int) *TradeTape {
	if capacity <= 0 {
		capacity = 1
	}
	return &TradeTape{
		buf:  make([]engine.TradeEvent, capacity),
		size: capacity,
	}
}

func (t *TradeTape) Append(tr engine.TradeEvent) {
	if t.count < t.size {
		t.buf[(t.start+t.count)%t.size] = tr
		t.count++
		return
	}
	// overwrite oldest
	t.buf[t.start] = tr
	t.start = (t.start + 1) % t.size
}

func (t *TradeTape) Last(n int) []engine.TradeEvent {
	if n <= 0 || t.count == 0 {
		return nil
	}
	if n > t.count {
		n = t.count
	}
	out := make([]engine.TradeEvent, n)
	// take last n in chronological order
	first := (t.start + (t.count - n)) % t.size
	for i := 0; i < n; i++ {
		out[i] = t.buf[(first+i)%t.size]
	}
	return out
}
