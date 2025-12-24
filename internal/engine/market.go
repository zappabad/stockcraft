package engine

import (
	"fmt"
)

type (
	TopicID string

	Ticker struct {
		ID       int64
		Name     string
		Decimals int8
	}

	Market struct {
		Orderbooks map[Ticker]*OrderBook
		Prices     map[Ticker]PriceTicks // current market prices for each ticker
		Tickers    []Ticker
	}
)

// NewMarket creates a basic market by initializing Orderbooks for each ticker provided.
func NewMarket(tickers []Ticker) Market {
	market := Market{
		Orderbooks: make(map[Ticker]*OrderBook),
		Prices:     make(map[Ticker]PriceTicks),
		Tickers:    tickers,
	}

	for _, t := range tickers {
		market.Orderbooks[Ticker(t)] = NewOrderBook()
	}

	return market
}

func (m *Market) GetOrderbook(ticker Ticker) (*OrderBook, error) {
	if ob, exists := m.Orderbooks[ticker]; exists {
		return ob, nil
	}
	return nil, fmt.Errorf("orderbook not found for ticker %s", ticker.Name)
}

// Returns best ask price for the ticker, an ok flag indicating if price exists, and error if any.
func (m *Market) GetPrice(ticker Ticker) (PriceTicks, bool, error) {
	orderbook, err := m.GetOrderbook(ticker)
	if err != nil {
		return 0, false, err
	}

	price, _, ok := engine.BestAsk()
	if !ok {
		// Not an error: just no asks right now.
		return 0, false, nil
	}

	// TODO: If Market is used concurrently, this map write needs a mutex.
	m.Prices[ticker] = price
	return price, true, nil
}

func (m *Market) GetTickers() []Ticker {
	return m.Tickers
}
