package engine

import (
	"fmt"
	"time"
)

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

// AddOrder adds a single order to the order book and attempts to match it.
// Returns any trades that occurred, the remaining order (also called a remainder, if any), and an error.
func (ob *OrderBook) AddOrder(o *Order) ([]Trade, *Order, error) {
    if o.Symbol != ob.Symbol {
        return nil, nil, fmt.Errorf("symbol mismatch")
    }

    switch o.Side {
    case Buy:
        return ob.addBuy(o)
    case Sell:
        return ob.addSell(o)
    default:
        return nil, nil, fmt.Errorf("unknown side")
    }
}

func (ob *OrderBook) addBuy(o *Order) ([]Trade, *Order, error) {
	ob.Bids.levels[o.Price] = append(ob.Bids.levels[o.Price], o)
	return nil, nil, nil
}

func (ob *OrderBook) addSell(o *Order) ([]Trade, *Order, error) {
	ob.Asks.levels[o.Price] = append(ob.Asks.levels[o.Price], o)
	return nil, nil, nil
}

func (ob *OrderBook) CheckTrade(o *Order) []Trade {
	if o.Side == Buy {
		for price := range ob.Asks.levels {
			if o.Price >= price {
				return true
			}
		}	
}


// TODO:  implement matching logic correctly.
func MatchOrders(ob *OrderBook) []Trade {
	var trades []Trade
	// Implement matching logic here using FIFO within price levels
	for price, buyOrders := range ob.Bids.levels {
		sellOrders, exists := ob.Asks.levels[price]
		if !exists {
			continue
		}
		buyIndex, sellIndex := 0, 0
		for buyIndex < len(buyOrders) && sellIndex < len(sellOrders) {
			buyOrder := buyOrders[buyIndex]
			sellOrder := sellOrders[sellIndex]
			tradeQty := min(buyOrder.Quantity, sellOrder.Quantity)
			trades = append(trades, Trade{
				BuyOrderID:  buyOrder.ID,
				SellOrderID: sellOrder.ID,
				Quantity:    tradeQty,
				Price:       price,
			})
			buyOrder.Quantity -= tradeQty
			sellOrder.Quantity -= tradeQty
			if buyOrder.Quantity == 0 {
				buyIndex++
			}
			if sellOrder.Quantity == 0 {
				sellIndex++
			}
		}
		// Remove filled orders
		ob.Bids.levels[price] = buyOrders[buyIndex:]
		ob.Asks.levels[price] = sellOrders[sellIndex:]
	}

	return trades
}	
// TODO: better comments
func (ob *OrderBook) AddOrders(orders []Order, m *Market) {
	for _, o := range orders {
		_, _, err := ob.AddOrder(&o)
		if err != nil {
			fmt.Printf("Error adding order %s: %v\n", o.ID, err)
		}
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
