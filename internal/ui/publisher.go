package ui

import "github.com/zappabad/stockcraft/internal/engine"

// UIChannelPublisher implements the engine.UIPublisher interface
type UIChannelPublisher struct {
	channels *UIChannels
}

// NewUIChannelPublisher creates a new publisher that writes to UI channels
func NewUIChannelPublisher(channels *UIChannels) *UIChannelPublisher {
	return &UIChannelPublisher{
		channels: channels,
	}
}

// PublishMarketUpdate sends market data to the UI
func (p *UIChannelPublisher) PublishMarketUpdate(tick int, prices map[string]float64, changes map[string]float64) {
	select {
	case p.channels.MarketUpdates <- MarketUpdateEvent{
		Tick:    tick,
		Prices:  copyMap(prices),
		Changes: copyMap(changes),
	}:
		// Event sent successfully
	default:
		// Channel is full, drop event to prevent blocking
		// In a production system, you might want to log this
	}
}

// PublishOrderUpdate sends order data to the UI
func (p *UIChannelPublisher) PublishOrderUpdate(tick int, orders []engine.Order) {
	select {
	case p.channels.OrderUpdates <- OrderUpdateEvent{
		Tick:   tick,
		Orders: copyOrders(orders),
	}:
		// Event sent successfully
	default:
		// Channel is full, drop event to prevent blocking
	}
}

// PublishNewsUpdate sends news data to the UI
func (p *UIChannelPublisher) PublishNewsUpdate(tick int, news *engine.News) {
	select {
	case p.channels.NewsUpdates <- NewsUpdateEvent{
		Tick: tick,
		News: copyNews(news),
	}:
		// Event sent successfully
	default:
		// Channel is full, drop event to prevent blocking
	}
}

// Helper functions to copy data structures to avoid race conditions

func copyMap(original map[string]float64) map[string]float64 {
	copy := make(map[string]float64)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

func copyOrders(original []engine.Order) []engine.Order {
	return append([]engine.Order{}, original...)
}

func copyNews(original *engine.News) *engine.News {
	if original == nil {
		return nil
	}

	copy := &engine.News{
		ID:       original.ID,
		Headline: original.Headline,
	}

	for _, impact := range original.SymbolImpacts {
		copy.SymbolImpacts = append(copy.SymbolImpacts, engine.SymbolImpact{
			Symbol: impact.Symbol,
			Impact: impact.Impact,
		})
	}

	return copy
}
