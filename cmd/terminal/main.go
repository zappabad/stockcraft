package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/zappabad/stockcraft/internal/engine"
	"github.com/zappabad/stockcraft/internal/ui"
)

func main() {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received interrupt signal, shutting down...")
		cancel()
	}()

	// 1. Create a basic market.
	market := engine.NewMarket()

	// 2. Create an order book.
	orderBook := engine.NewOrderBook()

	// 3. Create some traders.
	total_traders := 10
	traders := []engine.Trader{}

	for i := range total_traders {
		traderID := "trader-" + strconv.Itoa(i)
		traderSeed := rand.New(rand.NewSource(int64(i + 69420)))
		// Use both FOO and BAR symbols for more interesting trading
		traders = append(traders, engine.NewRandomTrader(traderID, []string{"FOO", "BAR", "AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "BRK.A", "NVDA", "JPM", "V", "JNJ", "WMT", "PG", "DIS"}, traderSeed))
	}

	// Simple sanity check: ensure we have at least one trader.
	if len(traders) == 0 {
		log.Fatal("no traders configured")
	}

	// 4. Generate News Engine
	newsEngine := engine.NewNewsEngine()

	// 5. Wire everything into a simulation.
	sim := engine.NewSimulation(market, orderBook, traders, newsEngine)

	// 6. Create UI channels and publisher
	uiChannels := ui.NewUIChannels()
	uiPublisher := ui.NewUIChannelPublisher(uiChannels)

	// 7. Start simulation in background
	go func() {
		log.Println("Starting simulation with 20ms tick rate...")
		sim.RunAsync(ctx, uiPublisher, 200*time.Millisecond)
	}()

	// 8. Start UI
	log.Println("Starting terminal UI...")
	if err := ui.RunUI(ctx, uiChannels); err != nil {
		log.Printf("UI error: %v", err)
	}

	log.Println("Application shut down successfully")
}
