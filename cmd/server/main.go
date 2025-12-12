package main

import (
	"log"
	"math/rand"
	"strconv"

	// Replace "yourmodule" with the module path from your go.mod.
	"github.com/zappabad/stockcraft/internal/engine"
)

func main() {
	var tickers []engine.Ticker = []engine.Ticker{"AAPL", "GOOGL", "NVDA", "BAR", "FOO", "BAZ"}

	// 1. Create a basic market.
	market := engine.NewMarket(tickers)

	// 2. Create an order book.
	orderBook := engine.NewOrderbook()

	// 3. Create some traders.
	// TODO: Replace these with real strategy types (frequent, swing, news-based).
	total_traders := 500
	traders := []engine.Trader{}

	for i := range total_traders {
		traderID := "trader-" + strconv.Itoa(i)
		traderSeed := rand.New(rand.NewSource(int64(i + 69420)))
		traders = append(traders, engine.NewRandomTrader(traderID, []string{"BAR"}, traderSeed))
	}

	// Simple sanity check: ensure we have at least one trader.
	if len(traders) == 0 {
		log.Fatal("no traders configured")
	}

	// 4. Generate News Engine
	newsEngine := engine.NewNewsEngine()

	// 4. Wire everything into a simulation.
	sim := engine.NewSimulation(market, orderBook, traders, newsEngine)

	// 5. Run for a few ticks and watch console output.
	// TODO: Make this configurable via flags or environment.
	sim.Run(0, 20)
}
