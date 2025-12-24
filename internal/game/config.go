package game

import (
	"time"

	brokerservice "github.com/zappabad/stockcraft/internal/broker/service"
	"github.com/zappabad/stockcraft/internal/market"
	marketservice "github.com/zappabad/stockcraft/internal/market/service"
	newsservice "github.com/zappabad/stockcraft/internal/news/service"
	"github.com/zappabad/stockcraft/internal/trader/runner"
)

// Config holds configuration for the game.
type Config struct {
	// Tickers is the list of tickers to create in the market.
	Tickers []market.Ticker
	// MarketConfig is the configuration for the market service.
	MarketConfig marketservice.Config
	// NewsConfig is the configuration for the news service.
	NewsConfig newsservice.Config
	// BrokerConfig is the configuration for the broker service.
	BrokerConfig brokerservice.Config
	// TraderConfigs is the configuration for each trader runner.
	TraderConfigs []runner.Config
	// EnableBroker determines whether the broker service is enabled.
	EnableBroker bool
}

// DefaultConfig returns a Config with reasonable defaults.
func DefaultConfig() Config {
	return Config{
		Tickers: []market.Ticker{
			{ID: 1, Name: "AAPL", Decimals: 2},
			{ID: 2, Name: "GOOGL", Decimals: 2},
			{ID: 3, Name: "MSFT", Decimals: 2},
		},
		MarketConfig: marketservice.DefaultConfig(),
		NewsConfig:   newsservice.DefaultConfig(),
		BrokerConfig: brokerservice.DefaultConfig(),
		TraderConfigs: []runner.Config{
			{
				TickInterval: 500 * time.Millisecond,
				EventBuffer:  256,
				DropEvents:   true,
			},
		},
		EnableBroker: true,
	}
}
