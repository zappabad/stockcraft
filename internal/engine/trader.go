package engine

import (
	"log"
	"math/rand"
)

// Trader is the behavior interface.
// Simulation will call Tick each "turn" and pass the current Market.
// Trader returns zero or more Orders it wants to place at this tick.
type Trader interface {
	ID() int64
	Tick(m Market) *Order
}

// RandomTrader is a dumb placeholder implementation.
// It demonstrates the flow without any real strategy.
type RandomTrader struct {
	id   int64
	seed *rand.Rand
}

// NewRandomTrader constructs a simple random trader.
// TODO: Replace this with more interesting strategies (frequent, swing, news-based, etc.).
func NewRandomTrader(id int64, symbols []string, seed *rand.Rand) *RandomTrader {
	return &RandomTrader{
		id:   id,
		seed: seed,
	}
}

func (t *RandomTrader) ID() int64 {
	return t.id
}

// Tick generates 0 or 1 random order per tick.
// This is intentionally stupid; it's just to show the wiring.
func (t *RandomTrader) Tick(m Market) (*Order, error) {

	// 50% chance to do nothing.
	if t.seed.Float64() < 0.5 {
		return nil, nil
	}

	side := SideBuy
	if t.seed.Float64() < 0.5 {
		side = SideSell
	}

	random_ticker := m.GetTickers()[t.seed.Intn(len(m.GetTickers()))]
	orderbook := m.Orderbooks[random_ticker]

	basePrice, err := m.GetPrice(random_ticker)
	if err != nil {
		log.Fatalf("failed to get price for ticker %s: %v", random_ticker, err)
	}

	// Randomly nudge around current price.
	price := basePrice * (0.95 + 0.1*t.seed.Float64()) // between 95% and 105%
	qty := t.seed.Intn(10) + 1                         // 1–10 units

	order := orderbook.NewLimitOrder(1, t.ID(), side, price, float64(qty))

	return order, nil
}
