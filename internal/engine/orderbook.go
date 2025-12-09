package engine

import (
	"fmt"
	"time"
)

// Order is the basic unit sent by traders to the order book.
// time-in-force, order type, etc.
type OrderID string
type Side int

const (
	BuySide  Side = 1
	SellSide Side = 2
)

type Order struct {
	ID        OrderID   // unique identifier
	Timestamp time.Time // when the order was created
	TraderID  string    // who sent the order
	Symbol    string    // instrument symbol, e.g. "AAPL"
	Side      Side      // buy or sell
	Quantity  int       // number of units
	Price     float64   // limit price; for now treat everything as limit orders
}

type Trade struct {
	BuyOrderID  OrderID
	SellOrderID OrderID
	Quantity    int
	Price       float64
}

// TODO: Add comments on what this actually is after removing placeholder
// SideBook represents one side of the order book (bids or asks).
// levels is a map from price to list of orders at that price.
type SideBook struct {
	levels map[float64][]*Order // map of price to list of orders at that price
	prices []float64            // sorted list of prices for quick access
}

type OrderBook struct {
	Symbol string
	Bids   SideBook           // buy side
	Asks   SideBook           // sell side
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

// AddOrder adds a single Order to the OrderBook.
// It doesn't check for matching; that comes later.
func (ob *OrderBook) AddOrder(o *Order) (*Order, error) {
	if o.Symbol != ob.Symbol {
		return nil, fmt.Errorf("symbol mismatch")
	}

	switch o.Side {
	case BuySide:
		return ob.addBuy(o)
	case SellSide:
		return ob.addSell(o)
	default:
		return nil, fmt.Errorf("unknown side")
	}
}

func (ob *OrderBook) addBuy(o *Order) (*Order, error) {
	slice_of_orders := ob.Bids.levels[o.Price] // Slice of current orders at a given price level
	slice_of_orders = append(slice_of_orders, o)
	return nil, nil
}

func (ob *OrderBook) addSell(o *Order) (*Order, error) {
	slice_of_orders := ob.Asks.levels[o.Price] // Slice of current orders at a given price level
	slice_of_orders = append(slice_of_orders, o)
	return nil, nil
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

func (ob *OrderBook) OrderMatching() []Order {
	return nil
}
