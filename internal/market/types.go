package market

// TickerID uniquely identifies a ticker.
type TickerID int64

// Ticker represents a tradeable instrument.
type Ticker struct {
	ID       int64
	Name     string
	Decimals int8
}

// TickerID returns the TickerID for this Ticker.
func (t Ticker) TickerID() TickerID {
	return TickerID(t.ID)
}
