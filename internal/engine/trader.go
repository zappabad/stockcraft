package engine

import (
	"math/rand"
)

// Trader is the behavior interface.
// Simulation will call Tick each "turn" and pass the current Market.
// Trader returns zero or more Orders it wants to place at this tick.
type Trader interface {
	ID() string
	Tick(m Market) []Order
}

// RandomTrader is a dumb placeholder implementation.
// It demonstrates the flow without any real strategy.
type RandomTrader struct {
	id      string
	symbols []string
	seed    *rand.Rand
}

// NewRandomTrader constructs a simple random trader.
// TODO: Replace this with more interesting strategies (frequent, swing, news-based, etc.).
func NewRandomTrader(id string, symbols []string, seed *rand.Rand) *RandomTrader {
	return &RandomTrader{
		id:      id,
		symbols: symbols,
		seed:    seed,
	}
}

func (t *RandomTrader) ID() string {
	return t.id
}

// Tick generates 0 or 1 random order per tick.
// This is intentionally stupid; it's just to show the wiring.
func (t *RandomTrader) Tick(m Market) []Order {
	// 50% chance to do nothing.
	if t.seed.Float64() < 0.5 {
		return nil
	}

	if len(t.symbols) == 0 {
		return nil
	}

	symbol := t.symbols[t.seed.Intn(len(t.symbols))]

	side := SideBuy
	if t.seed.Float64() < 0.5 {
		side = SideSell
	}

	basePrice := m.GetPrice(symbol)
	if basePrice == 0 {
		basePrice = 100 // fallback so the demo always has a price
	}

	// Randomly nudge around current price.
	price := basePrice * (0.95 + 0.1*t.seed.Float64()) // between 95% and 105%
	qty := t.seed.Intn(10) + 1                         // 1–10 units

	order := Order{
		TraderID: t.id,
		Symbol:   symbol,
		Side:     side,
		Quantity: qty,
		Price:    price,
	}

	return []Order{order}
}
