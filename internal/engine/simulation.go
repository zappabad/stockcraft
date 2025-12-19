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
		_, _, err := t.Tick(&s.Market)
		if err != nil {
			log.Fatalf("Trader %d failed to tick: %v\n", t.ID(), err)
		}
	}

	news := s.NewsEngine.GenerateNews(s.TickCount)
	if news != nil {
		fmt.Printf("News: %s\n", news.Details.Headline)
	}

	// Print a simple market snapshot to the console.
	// TODO: Replace with structured logging or a UI later.
	tickers := s.Market.GetTickers()

	for _, ticker := range tickers {
		price, err := s.Market.GetPrice(ticker)
		if err != nil {
			log.Printf("Error getting price for ticker %s: %v\n", ticker.Name, err)
		}
		fmt.Printf("Price[%s] = %d\n", ticker.Name, price)
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
