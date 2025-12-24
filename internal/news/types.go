package news

import "github.com/zappabad/stockcraft/internal/market"

// NewsID uniquely identifies a news item.
type NewsID int64

// NewsItem represents a news event.
type NewsItem struct {
	ID       NewsID
	Time     int64
	Ticker   market.TickerID // optional; 0 means market-wide news
	Headline string
	Body     string
	Severity int // 0=normal, positive=more severe/important
}
