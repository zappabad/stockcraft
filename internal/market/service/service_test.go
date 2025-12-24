package service

import (
	"context"
	"testing"
	"time"

	"github.com/zappabad/stockcraft/internal/market"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
)

func TestMarketServiceBasic(t *testing.T) {
	tickers := []market.Ticker{
		{ID: 1, Name: "AAPL", Decimals: 2},
		{ID: 2, Name: "GOOGL", Decimals: 2},
	}
	cfg := DefaultConfig()
	svc := NewMarketService(tickers, cfg)
	defer svc.Close()

	ctx := context.Background()

	// Submit order to first ticker
	report, err := svc.SubmitLimit(ctx, 1, 100, core.SideBuy, 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Remaining != 10 {
		t.Errorf("expected remaining 10, got %d", report.Remaining)
	}

	// Wait for view update
	time.Sleep(10 * time.Millisecond)

	// Check levels
	levels, err := svc.GetLevels(1, core.SideBuy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(levels) != 1 {
		t.Fatalf("expected 1 level, got %d", len(levels))
	}
	if levels[0].Price != 100 {
		t.Errorf("expected price 100, got %d", levels[0].Price)
	}

	// Unknown ticker should error
	_, err = svc.SubmitLimit(ctx, 999, 100, core.SideBuy, 100, 10)
	if err != ErrUnknownTicker {
		t.Errorf("expected ErrUnknownTicker, got %v", err)
	}
}

func TestMarketServiceSnapshot(t *testing.T) {
	tickers := []market.Ticker{
		{ID: 1, Name: "AAPL", Decimals: 2},
	}
	cfg := DefaultConfig()
	svc := NewMarketService(tickers, cfg)
	defer svc.Close()

	ctx := context.Background()

	// Submit bid and ask
	_, err := svc.SubmitLimit(ctx, 1, 100, core.SideBuy, 99, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = svc.SubmitLimit(ctx, 1, 200, core.SideSell, 101, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for view update
	time.Sleep(20 * time.Millisecond)

	// Check snapshot
	snap := svc.Snapshot()
	bp, ok := snap.ByTicker[1]
	if !ok {
		t.Fatal("ticker 1 not in snapshot")
	}
	if !bp.BidOK {
		t.Error("expected bid to exist")
	}
	if bp.BidPrice != 99 {
		t.Errorf("expected bid price 99, got %d", bp.BidPrice)
	}
	if !bp.AskOK {
		t.Error("expected ask to exist")
	}
	if bp.AskPrice != 101 {
		t.Errorf("expected ask price 101, got %d", bp.AskPrice)
	}
}

func TestMarketServiceTrade(t *testing.T) {
	tickers := []market.Ticker{
		{ID: 1, Name: "AAPL", Decimals: 2},
	}
	cfg := DefaultConfig()
	svc := NewMarketService(tickers, cfg)
	defer svc.Close()

	ctx := context.Background()

	// Submit sell limit
	_, err := svc.SubmitLimit(ctx, 1, 100, core.SideSell, 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Submit buy market to match
	report, err := svc.SubmitMarket(ctx, 1, 200, core.SideBuy, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(report.Fills))
	}

	// Wait for view update
	time.Sleep(20 * time.Millisecond)

	// Check last trade
	trades, err := svc.GetTradesLast(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Size != 5 {
		t.Errorf("expected trade size 5, got %d", trades[0].Size)
	}

	// Check snapshot has last trade
	snap := svc.Snapshot()
	bp := snap.ByTicker[1]
	if !bp.HasLast {
		t.Error("expected last trade in snapshot")
	}
	if bp.LastPrice != 100 {
		t.Errorf("expected last price 100, got %d", bp.LastPrice)
	}
}
