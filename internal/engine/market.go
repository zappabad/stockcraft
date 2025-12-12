package engine

type (
	Ticker string

	Market struct {
		Orderbooks map[Ticker]*Orderbook
		Prices     map[Ticker]float64
	}
)

// NewMarket creates a basic market by initializing Orderbooks for each ticker provided.
func NewMarket(tickers []string) Market {
	market := Market{
		Orderbooks: make(map[Ticker]*Orderbook),
		Prices:     make(map[Ticker]float64),
	}

	for _, t := range tickers {
		market.Orderbooks[Ticker(t)] = NewOrderbook()
		market.Prices[Ticker(t)] = 0.01 // default starting price
	}

	return market
}
