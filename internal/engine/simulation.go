package engine

import (
	"fmt"
	"log"
	"time"
)

// Simulation stitches together Market, Traders, and Orderbook.
// This is where your "game loop" lives.
type (
	Simulation struct {
		Market     Market
		Traders    []Trader
		NewsEngine *NewsEngine
		TickCount  int
	}
)

// NewSimulation wires up a new Simulation.
func NewSimulation(m Market, traders []Trader, ne *NewsEngine) *Simulation {
	return &Simulation{
		Market:     m,
		Traders:    traders,
		NewsEngine: ne,
	}
}

// Step runs a single tick:
//   - Ask each trader for orders.
//   - Send those orders into the order book.
//   - Let the order book update the market (for now, naively).
func (s *Simulation) Step() {
	s.TickCount++
	fmt.Printf("\n=== Tick %d ===\n", s.TickCount)
	for _, t := range s.Traders {
		_, _, err := t.Tick(s.TickCount, &s.Market)
		if err != nil {
			log.Fatalf("Trader %d failed to tick: %v\n", t.ID(), err)
		}
	}

	news := s.NewsEngine.GenerateNews(s.TickCount)
	if news != nil {
		fmt.Printf("News: %s\n", news.Details.Headline)
	}

	// Print a simple market snapshot to the console.
	printMarketSnapshot(&s.Market)

	// Inspect orderbooks (for debugging).
	for _, ticker := range s.Market.GetTickers() {
		orderbook, err := s.Market.GetOrderbook(ticker)
		if err != nil {
			log.Printf("Error getting orderbook for ticker %s: %v\n", ticker.Name, err)
			continue
		}
		fmt.Printf("Orderbook for %s:\n%s\n", ticker.Name, orderbook.String())
	}

}

func printMarketSnapshot(m *Market) {
	tickers := m.GetTickers()
	for _, ticker := range tickers {
		price, ok, err := m.GetPrice(ticker)
		if err != nil {
			log.Printf("Error getting price for ticker %s: %v\n", ticker.Name, err)
			continue
		}
		if ok {
			fmt.Printf("Ticker: %s, Price: %d\n", ticker.Name, price)
		} else {
			fmt.Printf("Ticker: %s, Price: N/A\n", ticker.Name)
		}
	}
}

// Run executes N ticks synchronously.
// For the first prototype, this is fine; later you may want a real-time loop.
func (s *Simulation) Run(ticks int, tick_rate int) {
	if ticks > 0 {
		for i := 0; i < ticks; i++ {
			time.Sleep(time.Duration(1000/tick_rate) * time.Millisecond)
			s.Step()
		}
	} else {
		for {
			time.Sleep(time.Duration(1000/tick_rate) * time.Millisecond)
			s.Step()
		}
	}
}
