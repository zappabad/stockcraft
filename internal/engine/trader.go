package engine

import (
	"fmt"
	"log"
	"math/rand"
)

// Trader is the behavior interface.
// Simulation will call Tick each "turn" and pass the current Market.
// Trader returns zero or more Orders it wants to place at this tick.
type Trader interface {
	ID() int64
	Tick(tick_n int, m *Market) (*Order, []Match, error)
}

// RandomTrader is a dumb placeholder implementation.
// It demonstrates the flow without any real strategy.
type RandomTrader struct {
	id      int64
	seed    *rand.Rand
	zipf    *rand.Zipf
	tickers []Ticker
}

// NewRandomTrader constructs a simple random trader.
// TODO: Replace this with more interesting strategies (frequent, swing, news-based, etc.).
func NewRandomTrader(id int64, tickers []Ticker, seed *rand.Rand) *RandomTrader {
	return &RandomTrader{
		id:      id,
		seed:    seed,
		zipf:    rand.NewZipf(seed, 1.15, 3, 20), // s, v, imax (tune)
		tickers: tickers,
	}
}

func (t *RandomTrader) ID() int64 {
	return t.id
}

// Tick generates 0 or 1 random order per tick.
// This is intentionally stupid; it's just to show the wiring.
func (t *RandomTrader) Tick(tick_n int, m *Market) (*Order, []Match, error) {

	// 50% chance to do nothing.
	if t.seed.Float64() < 0.5 {
		return nil, nil, nil
	}

	// random_ticker := m.GetTickers()[t.seed.Intn(len(m.GetTickers()))]
	random_ticker := t.tickers[t.seed.Intn(len(t.tickers))]

	orderbook, err := m.GetOrderbook(random_ticker)

	if err != nil {
		log.Fatalf("failed to get orderbook for ticker %s: %v", random_ticker.Name, err)
	}

	basePrice, ok, err := m.GetPrice(random_ticker)
	if err != nil {
		log.Fatalf("failed to get price for ticker %s: %v", random_ticker.Name, err)
	}
	if !ok { // No price yet; use a fake base price.
		fmt.Printf("No price for ticker %s; using base price.\n", random_ticker.Name)
		basePrice = PriceTicks(100)
	}

	var price PriceTicks
	qty := int64(1)

	side := SideBuy
	if t.seed.Float64() < 0.5 {
		side = SideSell
	}
	// Randomly nudge around current price.
	delta := PriceTicks(5 * t.seed.Intn(2)) // 0..1 ticks
	if side == SideBuy {
		price = basePrice - delta
	} else {
		price = basePrice + delta
	}

	order := NewLimitOrder(t.ID(), side, price, Size(qty))
	matches, _, err := engine.SubmitLimitOrder(order)
	if err != nil {
		log.Fatalf("failed to submit order: %v", err)
	}
	return order, matches, nil
}
