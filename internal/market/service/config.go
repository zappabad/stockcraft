package service

import (
	orderbookservice "github.com/zappabad/stockcraft/internal/orderbook/service"
)

// Config holds configuration for the market service.
type Config struct {
	// Book is the configuration for each orderbook service.
	Book orderbookservice.Config
	// MarketEventBuffer is the size of the consolidated market events channel.
	MarketEventBuffer int
	// DropMarketEvents determines whether the market events channel drops on overflow.
	DropMarketEvents bool
}

// DefaultConfig returns a Config with reasonable defaults.
func DefaultConfig() Config {
	return Config{
		Book:              orderbookservice.DefaultConfig(),
		MarketEventBuffer: 1024,
		DropMarketEvents:  true,
	}
}
