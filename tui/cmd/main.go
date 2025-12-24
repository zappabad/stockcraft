package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zappabad/stockcraft/internal/game"
	"github.com/zappabad/stockcraft/internal/market"
	marketservice "github.com/zappabad/stockcraft/internal/market/service"
	"github.com/zappabad/stockcraft/internal/news"
	newsservice "github.com/zappabad/stockcraft/internal/news/service"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
	"github.com/zappabad/stockcraft/tui"
)

func main() {
	// Create game configuration
	cfg := game.DefaultConfig()
	cfg.Tickers = []market.Ticker{
		{ID: 1, Name: "AAPL", Decimals: 2},
		{ID: 2, Name: "GOOGL", Decimals: 2},
		{ID: 3, Name: "MSFT", Decimals: 2},
		{ID: 4, Name: "AMZN", Decimals: 2},
		{ID: 5, Name: "TSLA", Decimals: 2},
	}

	// Create market service
	marketService := marketservice.NewMarketService(cfg.Tickers, cfg.MarketConfig)
	defer marketService.Close()

	// Create news service
	newsService := newsservice.NewNewsService(cfg.NewsConfig)
	defer newsService.Close()

	// Seed some initial orders to create a market
	seedMarket(marketService, cfg.Tickers)

	// Publish some initial news
	seedNews(newsService)

	// Start background trading simulation
	go simulateTrading(marketService, cfg.Tickers)
	go simulateNews(newsService)

	// Create and run TUI
	playerUserID := core.UserID(1000) // Player's user ID
	model := tui.NewModel(marketService, newsService, playerUserID)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func seedMarket(marketService *marketservice.MarketService, tickers []market.Ticker) {
	ctx := context.Background()
	userID := core.UserID(1)

	// Base prices for each ticker
	basePrices := map[string]int64{
		"AAPL":  17500, // $175.00
		"GOOGL": 14000, // $140.00
		"MSFT":  37500, // $375.00
		"AMZN":  17800, // $178.00
		"TSLA":  25000, // $250.00
	}

	for _, ticker := range tickers {
		basePrice := basePrices[ticker.Name]
		if basePrice == 0 {
			basePrice = 10000
		}

		tid := ticker.TickerID()

		// Place several bid orders (buy)
		for i := 0; i < 5; i++ {
			price := core.PriceTicks(basePrice - int64(i*50) - int64(i%3)*10)
			size := core.Size(100 + i*50)
			marketService.SubmitLimit(ctx, tid, userID, core.SideBuy, price, size)
		}

		// Place several ask orders (sell)
		for i := 0; i < 5; i++ {
			price := core.PriceTicks(basePrice + int64(i*50) + int64(i%3)*10)
			size := core.Size(100 + i*50)
			marketService.SubmitLimit(ctx, tid, userID, core.SideSell, price, size)
		}
	}
}

func seedNews(newsService *newsservice.NewsService) {
	headlines := []string{
		"Markets open higher amid positive economic data",
		"Tech sector shows strong momentum in early trading",
		"Federal Reserve signals continued focus on inflation",
		"Quarterly earnings season kicks off this week",
	}

	for _, headline := range headlines {
		newsService.Publish(news.NewsItem{
			Headline: headline,
			Severity: 0,
		})
	}
}

func simulateTrading(marketService *marketservice.MarketService, tickers []market.Ticker) {
	ctx := context.Background()
	traderID := core.UserID(999)

	for {
		time.Sleep(500 * time.Millisecond)

		// Pick a random ticker
		ticker := tickers[time.Now().UnixNano()%int64(len(tickers))]
		tid := ticker.TickerID()

		// Get current levels
		bids, _ := marketService.GetLevels(tid, core.SideBuy)
		asks, _ := marketService.GetLevels(tid, core.SideSell)

		if len(bids) == 0 || len(asks) == 0 {
			continue
		}

		// Simulate some trading activity
		action := time.Now().UnixNano() % 10

		switch {
		case action < 3:
			// Place a new bid slightly below best
			price := bids[0].Price - core.PriceTicks(time.Now().UnixNano()%30)
			size := core.Size(50 + time.Now().UnixNano()%100)
			marketService.SubmitLimit(ctx, tid, traderID, core.SideBuy, price, size)

		case action < 6:
			// Place a new ask slightly above best
			price := asks[0].Price + core.PriceTicks(time.Now().UnixNano()%30)
			size := core.Size(50 + time.Now().UnixNano()%100)
			marketService.SubmitLimit(ctx, tid, traderID, core.SideSell, price, size)

		case action < 8:
			// Market buy
			size := core.Size(10 + time.Now().UnixNano()%50)
			marketService.SubmitMarket(ctx, tid, traderID, core.SideBuy, size)

		default:
			// Market sell
			size := core.Size(10 + time.Now().UnixNano()%50)
			marketService.SubmitMarket(ctx, tid, traderID, core.SideSell, size)
		}
	}
}

func simulateNews(newsService *newsservice.NewsService) {
	newsHeadlines := []string{
		"Breaking: Major acquisition announced in tech sector",
		"Analyst upgrades rating on leading semiconductor company",
		"Economic indicators suggest continued growth",
		"Supply chain improvements boost manufacturing outlook",
		"Consumer spending remains strong despite inflation concerns",
		"Central bank maintains current policy stance",
		"Corporate buybacks reach record levels",
		"New product launches drive optimism in retail",
		"Energy prices stabilize after recent volatility",
		"International trade tensions ease on diplomatic progress",
		"Earnings beat expectations across multiple sectors",
		"Infrastructure spending bill gains momentum",
		"Housing market shows signs of cooling",
		"Job market remains tight as wages increase",
		"Technology companies lead market gains",
	}

	importantHeadlines := []string{
		"BREAKING: Fed announces surprise rate decision",
		"ALERT: Major company reports earnings miss",
		"URGENT: Regulatory investigation announced",
		"FLASH: Merger deal falls through",
	}

	idx := 0
	for {
		time.Sleep(time.Duration(3+time.Now().UnixNano()%5) * time.Second)

		var headline string
		var severity int

		// Occasionally publish important news
		if time.Now().UnixNano()%10 == 0 {
			headline = importantHeadlines[time.Now().UnixNano()%int64(len(importantHeadlines))]
			severity = 1
		} else {
			headline = newsHeadlines[idx%len(newsHeadlines)]
			severity = 0
			idx++
		}

		newsService.Publish(news.NewsItem{
			Headline: headline,
			Severity: severity,
		})
	}
}
