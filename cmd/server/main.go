package main

import (
	"log"
	"math/rand"

	// Replace "yourmodule" with the module path from your go.mod.
	"github.com/zappabad/stockcraft/internal/engine"
)

func main() {
	tickers := []engine.Ticker{
		{Name: "AAPL", Decimals: 2},
		{Name: "GOOGL", Decimals: 2},
		{Name: "NVDA", Decimals: 2},
		{Name: "AMZN", Decimals: 2},
		{Name: "MSFT", Decimals: 2},
		{Name: "TSLA", Decimals: 2},
		{Name: "META", Decimals: 2},
		{Name: "NFLX", Decimals: 2},
		{Name: "BABA", Decimals: 2},
		{Name: "INTC", Decimals: 2},
	}

	// 1. Create a basic market.
	market := engine.NewMarket(tickers)

	// 2. Create some traders.
	// TODO: Replace these with real strategy types (frequent, swing, news-based).
	var total_traders int64 = 500
	traders := []engine.Trader{}

	for i := range total_traders {
		traderSeed := rand.New(rand.NewSource(int64(i + 69420)))
		traders = append(traders, engine.NewRandomTrader(i, []string{"BAR"}, traderSeed))
	}

	// Simple sanity check: ensure we have at least one trader.
	if len(traders) == 0 {
		log.Fatal("no traders configured")
	}

	// 4. Generate News Engine
	newsEngine := engine.NewNewsEngine()

	// 4. Wire everything into a simulation.
	sim := engine.NewSimulation(market, traders, newsEngine)

	// 5. Run for a few ticks and watch console output.
	// TODO: Make this configurable via flags or environment.
	sim.Run(0, 20)
}
