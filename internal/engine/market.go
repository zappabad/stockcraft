package engine

// Market holds current prices for each symbol.
// For the first prototype, this is just a simple map you mutate.
type Market struct {
	Prices map[string]float64
}

// NewMarket creates a basic market with some starter prices.
func NewMarket() Market {
	return Market{
		Prices: map[string]float64{
			"FOO": 100.0,
			"BAR": 50.0,
		},
	}
}

// GetPrice returns the current price for a symbol.
// TODO: Decide what to do if symbol doesn't exist (error vs default).
func (m Market) GetPrice(symbol string) float64 {
	price, ok := m.Prices[symbol]
	if !ok {
		// For now, just return 0 for unknown symbols.
		// You probably want to handle this more strictly later.
		return 0
	}
	return price
}

// SetPrice sets the current price for a symbol.
// TODO: Integrate this with a real matching engine later.
func (m *Market) SetPrice(symbol string, price float64) {
	m.Prices[symbol] = price
}
