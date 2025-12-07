package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Simulation stitches together Market, Traders, and OrderBook.
// This is where your "game loop" lives.
type Simulation struct {
	Market     Market
	OrderBook  *OrderBook
	Traders    []Trader
	NewsEngine *NewsEngine
	TickCount  int
	mu         sync.RWMutex
}

// UIChannels interface for sending updates to UI
type UIPublisher interface {
	PublishMarketUpdate(tick int, prices map[string]float64, changes map[string]float64)
	PublishOrderUpdate(tick int, orders []Order)
	PublishNewsUpdate(tick int, news *News)
}

// NewSimulation wires up a new Simulation.
func NewSimulation(m Market, ob *OrderBook, traders []Trader, ne *NewsEngine) *Simulation {
	return &Simulation{
		Market:     m,
		OrderBook:  ob,
		Traders:    traders,
		NewsEngine: ne,
	}
}

// Step runs a single tick:
//   - Ask each trader for orders.
//   - Send those orders into the order book.
//   - Let the order book update the market (for now, naively).
func (s *Simulation) Step() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TickCount++
	fmt.Printf("\n=== Tick %d ===\n", s.TickCount)

	var allOrders []Order

	for _, t := range s.Traders {
		orders := t.Tick(s.Market)
		if len(orders) == 0 {
			continue
		}
		allOrders = append(allOrders, orders...)
	}

	if len(allOrders) == 0 {
		fmt.Println("No orders this tick.")
		return
	}

	news := s.NewsEngine.GenerateNews(s.TickCount)
	if news != nil {
		fmt.Printf("News: %s\n", news.Headline)
	}

	s.OrderBook.ApplyOrders(allOrders, &s.Market)

	// Print a simple market snapshot to the console.
	// TODO: Replace with structured logging or a UI later.
	for _, symbol := range s.Market.GetOrderedSymbols() {
		if price, exists := s.Market.Prices[symbol]; exists {
			fmt.Printf("Price[%s] = %.2f\n", symbol, price)
		}
	}
}

// StepWithUI runs a single tick and publishes updates to UI
func (s *Simulation) StepWithUI(publisher UIPublisher) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store previous prices for change calculation
	prevPrices := make(map[string]float64)
	for _, symbol := range s.Market.GetOrderedSymbols() {
		if price, exists := s.Market.Prices[symbol]; exists {
			prevPrices[symbol] = price
		}
	}

	s.TickCount++

	var allOrders []Order

	for _, t := range s.Traders {
		orders := t.Tick(s.Market)
		if len(orders) == 0 {
			continue
		}
		allOrders = append(allOrders, orders...)
	}

	news := s.NewsEngine.GenerateNews(s.TickCount)

	s.OrderBook.ApplyOrders(allOrders, &s.Market)

	// Calculate price changes
	changes := make(map[string]float64)
	for _, symbol := range s.Market.GetOrderedSymbols() {
		if price, exists := s.Market.Prices[symbol]; exists {
			if prevPrice, prevExists := prevPrices[symbol]; prevExists {
				changes[symbol] = price - prevPrice
			} else {
				changes[symbol] = 0
			}
		}
	}

	// Publish updates to UI
	publisher.PublishMarketUpdate(s.TickCount, s.Market.Prices, changes)

	if len(allOrders) > 0 {
		publisher.PublishOrderUpdate(s.TickCount, allOrders)
	}

	if news != nil {
		publisher.PublishNewsUpdate(s.TickCount, news)
	}
}

// GetTickCount returns current tick count (thread-safe)
func (s *Simulation) GetTickCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TickCount
}

// GetMarketSnapshot returns current market state (thread-safe)
func (s *Simulation) GetMarketSnapshot() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]float64)
	for _, symbol := range s.Market.GetOrderedSymbols() {
		if price, exists := s.Market.Prices[symbol]; exists {
			snapshot[symbol] = price
		}
	}
	return snapshot
}

// Run executes N ticks synchronously.
// For the first prototype, this is fine; later you may want a real-time loop.
func (s *Simulation) Run(ticks int) {
	for i := 0; i < ticks; i++ {
		s.Step()
	}
}

// RunAsync executes the simulation in real-time with UI updates
func (s *Simulation) RunAsync(ctx context.Context, publisher UIPublisher, tickInterval time.Duration) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.StepWithUI(publisher)
		}
	}
}
