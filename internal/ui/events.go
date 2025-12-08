package ui

import (
	"time"

	"github.com/zappabad/stockcraft/internal/engine"
)

// UI event types sent from simulation engine to UI
type UIEvent interface {
	Type() string
}

// MarketUpdateEvent contains price changes for display
type MarketUpdateEvent struct {
	Tick    int
	Prices  map[string]float64
	Changes map[string]float64 // price change from previous tick
}

func (e MarketUpdateEvent) Type() string { return "market_update" }

// OrderUpdateEvent contains new orders for display
type OrderUpdateEvent struct {
	Tick   int
	Orders []engine.Order
}

func (e OrderUpdateEvent) Type() string { return "order_update" }

// NewsUpdateEvent contains news for ticker display
type NewsUpdateEvent struct {
	Tick int
	News *engine.News
}

func (e NewsUpdateEvent) Type() string { return "news_update" }

// StockSelectionEvent contains selected stock symbol for filtering
type StockSelectionEvent struct {
	Symbol string
}

func (e StockSelectionEvent) Type() string { return "stock_selection" }

// UIChannels holds all communication channels between simulation and UI
type UIChannels struct {
	MarketUpdates   chan MarketUpdateEvent
	OrderUpdates    chan OrderUpdateEvent
	NewsUpdates     chan NewsUpdateEvent
	StockSelections chan StockSelectionEvent
	Shutdown        chan struct{}
}

// NewUIChannels creates channels with appropriate buffer sizes for 20ms ticks
func NewUIChannels() *UIChannels {
	return &UIChannels{
		// 50 capacity = 1 second worth of events at 20ms per tick
		MarketUpdates: make(chan MarketUpdateEvent, 50),
		OrderUpdates:  make(chan OrderUpdateEvent, 50),
		// 10 capacity for news (lower frequency)
		NewsUpdates: make(chan NewsUpdateEvent, 10),
		// Small buffer for stock selections (user interaction)
		StockSelections: make(chan StockSelectionEvent, 5),
		Shutdown:        make(chan struct{}),
	}
}

// TickData aggregates all data from a single simulation tick
type TickData struct {
	Tick   int
	Time   time.Time
	Prices map[string]float64
	Orders []engine.Order
	News   *engine.News
}
