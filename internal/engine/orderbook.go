package engine

// OrderBook is a placeholder for the matching engine.
// For now it just stores orders and naively updates Market prices.
//
// TODO:
//   - Implement a real bid/ask book per symbol.
//   - Match orders (price-time priority).
//   - Output trades, PnL, etc.
type OrderBook struct {
	orders []Order
}

// NewOrderBook constructs an empty order book.
func NewOrderBook() *OrderBook {
	return &OrderBook{
		orders: make([]Order, 0),
	}
}

// ApplyOrders takes a batch of new orders for a single tick.
// Right now it:
//   - Appends them to the in-memory slice.
//   - Updates the market price to the last order price per symbol.
//
// This is just to make the system visibly "do something".
func (ob *OrderBook) ApplyOrders(orders []Order, m *Market) {
	for _, o := range orders {
		ob.orders = append(ob.orders, o)

		// Naive "last trade wins" price update.
		m.SetPrice(o.Symbol, o.Price)

		// fmt.Printf(
		// 	"Applied order: trader=%s symbol=%s side=%v qty=%d price=%.2f\n",
		// 	o.TraderID, o.Symbol, o.Side, o.Quantity, o.Price,
		// )
	}
}

// SnapshotOrders returns a copy of the internal orders slice.
//
// TODO: Replace this with a richer view (e.g., per-symbol best bid/ask)
// or remove it if you don't need it.
func (ob *OrderBook) SnapshotOrders() []Order {
	out := make([]Order, len(ob.orders))
	copy(out, ob.orders)
	return out
}
