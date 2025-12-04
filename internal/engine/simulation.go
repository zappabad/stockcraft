package engine

import "fmt"

// Simulation stitches together Market, Traders, and OrderBook.
// This is where your "game loop" lives.
type Simulation struct {
	Market    Market
	OrderBook *OrderBook
	Traders   []Trader
	TickCount int
}

// NewSimulation wires up a new Simulation.
func NewSimulation(m Market, ob *OrderBook, traders []Trader) *Simulation {
	return &Simulation{
		Market:    m,
		OrderBook: ob,
		Traders:   traders,
	}
}

// Step runs a single tick:
//   - Ask each trader for orders.
//   - Send those orders into the order book.
//   - Let the order book update the market (for now, naively).
func (s *Simulation) Step() {
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

	s.OrderBook.ApplyOrders(allOrders, &s.Market)

	// Print a simple market snapshot to the console.
	// TODO: Replace with structured logging or a UI later.
	for symbol, price := range s.Market.Prices {
		fmt.Printf("Price[%s] = %.2f\n", symbol, price)
	}
}

// Run executes N ticks synchronously.
// For the first prototype, this is fine; later you may want a real-time loop.
func (s *Simulation) Run(ticks int) {
	for i := 0; i < ticks; i++ {
		s.Step()
	}
}
