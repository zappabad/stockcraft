package engine

import (
	"fmt"
)

type (
	Ticker struct {
		Name     string
		Decimals int8
	}

	TopicID string

	Market struct {
		Orderbooks map[Ticker]*OrderBook
		Prices     map[Ticker]float64 // current market prices for each ticker
		Tickers    []Ticker
	}
)

// NewMarket creates a basic market by initializing Orderbooks for each ticker provided.
func NewMarket(tickers []Ticker) Market {
	market := Market{
		Orderbooks: make(map[Ticker]*OrderBook),
		Prices:     make(map[Ticker]float64),
		Tickers:    tickers,
	}

	for _, t := range tickers {
		market.Orderbooks[Ticker(t)] = NewOrderBook()
		market.Prices[Ticker(t)] = 0.01 // default starting price
	}

	return market
}

func (m *Market) GetPrice(ticker Ticker) (float64, error) {
	if price, exists := m.Prices[ticker]; exists {
		return price, nil
	}
	return 0.0, fmt.Errorf("price not found for ticker %s", ticker)
}

func (m *Market) SetPrice(ticker Ticker, price float64) error {
	if _, exists := m.Prices[ticker]; exists {
		m.Prices[ticker] = price
		return nil
	}
	return fmt.Errorf("cannot set price for unknown ticker %s", ticker)
}

func (m *Market) GetOrderbook(ticker Ticker) (*OrderBook, error) {
	if ob, exists := m.Orderbooks[ticker]; exists {
		return ob, nil
	}
	return nil, fmt.Errorf("orderbook not found for ticker %s", ticker)
}

func (m *Market) GetTickers() []Ticker {
	return m.Tickers
}
