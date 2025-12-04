package main

import (
	"log"
	"math/rand"

	// Replace "yourmodule" with the module path from your go.mod.
	"github.com/zappabad/stockcraft/internal/engine"
)

func main() {
	// Seed RNG once at startup for the random traders.
	random_seed := rand.New(rand.NewSource(420)) // for reproducibility during testing

	// 1. Create a basic market.
	market := engine.NewMarket()

	// 2. Create an order book.
	orderBook := engine.NewOrderBook()

	// 3. Create some traders.
	// TODO: Replace these with real strategy types (frequent, swing, news-based).
	traders := []engine.Trader{
		engine.NewRandomTrader("trader-1", []string{"FOO", "BAR"}, random_seed),
		engine.NewRandomTrader("trader-2", []string{"FOO"}, random_seed),
		engine.NewRandomTrader("trader-3", []string{"FOO", "BAR"}, random_seed),
		engine.NewRandomTrader("trader-4", []string{"FOO", "BAR"}, random_seed),
	}

	// Simple sanity check: ensure we have at least one trader.
	if len(traders) == 0 {
		log.Fatal("no traders configured")
	}

	// 4. Wire everything into a simulation.
	sim := engine.NewSimulation(market, orderBook, traders)

	// 5. Run for a few ticks and watch console output.
	// TODO: Make this configurable via flags or environment.
	sim.Run(10)
}
