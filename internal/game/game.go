package game

import (
	"sync"

	brokerservice "github.com/zappabad/stockcraft/internal/broker/service"
	marketservice "github.com/zappabad/stockcraft/internal/market/service"
	newsservice "github.com/zappabad/stockcraft/internal/news/service"
	"github.com/zappabad/stockcraft/internal/trader"
	"github.com/zappabad/stockcraft/internal/trader/runner"
	"github.com/zappabad/stockcraft/internal/trader/strategy"
)

// Game owns all the game subsystems and manages their lifecycle.
type Game struct {
	Market  *marketservice.MarketService
	News    *newsservice.NewsService
	Traders []*runner.Runner
	Broker  *brokerservice.BrokerService

	cfg Config
	mu  sync.Mutex
}

// NewGame creates a new Game with the given configuration.
func NewGame(cfg Config) *Game {
	g := &Game{cfg: cfg}

	// Create market service
	g.Market = marketservice.NewMarketService(cfg.Tickers, cfg.MarketConfig)

	// Create news service
	g.News = newsservice.NewNewsService(cfg.NewsConfig)

	// Create broker service if enabled
	if cfg.EnableBroker {
		g.Broker = brokerservice.NewBrokerService(cfg.BrokerConfig)
	}

	// Create traders
	for i, tcfg := range cfg.TraderConfigs {
		traderID := trader.TraderID(i + 1)
		strat := strategy.NewExampleStrategy(traderID)

		r := runner.NewRunner(
			tcfg,
			traderID,
			strat,
			g.Market, // MarketReader
			g.News,   // NewsReader
			g.Market, // OrderSender
		)
		g.Traders = append(g.Traders, r)

		// Attach trader events to broker if enabled
		if g.Broker != nil {
			g.Broker.AttachTraderEvents(r.Events())
		}
	}

	// Attach news events to broker if enabled
	if g.Broker != nil {
		g.Broker.AttachNewsEvents(g.News.Events())
	}

	return g
}

// Close shuts down all game subsystems in reverse dependency order.
func (g *Game) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Stop traders first
	for _, t := range g.Traders {
		t.Close()
	}

	// Stop news
	if g.News != nil {
		g.News.Close()
	}

	// Stop market
	if g.Market != nil {
		g.Market.Close()
	}

	// Stop broker last
	if g.Broker != nil {
		g.Broker.Close()
	}
}
