package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/zappabad/stockcraft/internal/orderbook/core"
)

func TestServiceBasic(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg)
	defer svc.Close()

	ctx := context.Background()

	// Submit a limit order
	report, err := svc.SubmitLimit(ctx, 1, core.SideBuy, 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Remaining != 10 {
		t.Errorf("expected remaining 10, got %d", report.Remaining)
	}
	if !report.Rested {
		t.Error("expected order to rest")
	}

	// Check view is updated
	time.Sleep(10 * time.Millisecond) // wait for event dispatcher
	levels := svc.GetLevels(core.SideBuy)
	if len(levels) != 1 {
		t.Fatalf("expected 1 level, got %d", len(levels))
	}
	if levels[0].Price != 100 {
		t.Errorf("expected price 100, got %d", levels[0].Price)
	}
	if levels[0].Size != 10 {
		t.Errorf("expected size 10, got %d", levels[0].Size)
	}
}

func TestServiceConcurrent(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg)
	defer svc.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Submit many orders concurrently
	numOrders := 100
	wg.Add(numOrders)
	for i := 0; i < numOrders; i++ {
		go func(i int) {
			defer wg.Done()
			price := core.PriceTicks(100 + i%10)
			_, err := svc.SubmitLimit(ctx, core.UserID(i+1), core.SideBuy, price, 1)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify all orders are visible in view
	orders := svc.GetOrders(core.SideBuy)
	if len(orders) != numOrders {
		t.Errorf("expected %d orders, got %d", numOrders, len(orders))
	}
}

func TestServiceCancel(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg)
	defer svc.Close()

	ctx := context.Background()

	// Submit a limit order
	report, err := svc.SubmitLimit(ctx, 1, core.SideBuy, 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for view update
	time.Sleep(10 * time.Millisecond)

	// Cancel the order
	cancelReport, err := svc.Cancel(ctx, report.OrderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cancelReport.CanceledSize != 10 {
		t.Errorf("expected canceled size 10, got %d", cancelReport.CanceledSize)
	}

	// Wait for view update
	time.Sleep(10 * time.Millisecond)

	// Verify order is removed
	orders := svc.GetOrders(core.SideBuy)
	if len(orders) != 0 {
		t.Errorf("expected 0 orders, got %d", len(orders))
	}
}

func TestServiceEvents(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg)
	defer svc.Close()

	ctx := context.Background()

	// Subscribe to events
	events := svc.Events()

	// Submit a limit order
	_, err := svc.SubmitLimit(ctx, 1, core.SideBuy, 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read event
	select {
	case ev := <-events:
		if _, ok := ev.(core.OrderRestedEvent); !ok {
			t.Errorf("expected OrderRestedEvent, got %T", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}
