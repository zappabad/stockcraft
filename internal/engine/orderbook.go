package engine

import "time"

// Order is the basic unit sent by traders to the order book.
// time-in-force, order type, etc.
type OrderID string

type Order struct {
	ID		  OrderID   // unique identifier
	Timestamp time.Time // when the order was created
	TraderID  string    // who sent the order
	Symbol    string    // instrument symbol, e.g. "AAPL"
	Side      Side      // buy or sell
	Quantity  int       // number of units
	Price     float64   // limit price; for now treat everything as limit orders
}

// TODO: Add comments on what this actually is after removing placeholder
type SideBook struct {
	levels map[float64][]*Order // map of price to list of orders at that price
	prices []float64          	// sorted list of prices for quick access
}

type OrderBook struct {
	Symbol  string
	Bids   SideBook 		  // buy side
	Asks   SideBook 		  // sell side
	byID   map[OrderID]*Order // for cancel/replace TODO: (idk what this means replace it)
}

// NewOrderBook constructs an empty order book.
func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids: SideBook{
			levels: make(map[float64][]*Order),
			prices: []float64{},
		},
		Asks: SideBook{
			levels: make(map[float64][]*Order),
			prices: []float64{},
		},
	}
}

func (ob *OrderBook) AddOrder(order *Order) (trades []Trade, resting *Order, err error) {
	// TODO: implement order matching logic
	return nil, nil, nil
}

// TODO: better comments
func (ob *OrderBook) AddOrders(orders []Order, m *Market) {
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

func (ob *OrderBook) OrderMatching() []Order {

// SnapshotOrders returns a copy of the internal orders slice.
//
// TODO: Replace this with a richer view (e.g., per-symbol best bid/ask)
// or remove it if you don't need it.
func (ob *OrderBook) SnapshotOrders() []Order {
	out := make([]Order, len(ob.orders))
	copy(out, ob.orders)
	return out
}
