package core

import (
	"testing"
)

func TestSubmitLimit(t *testing.T) {
	c := NewCore()

	// Submit a buy limit order
	order := Order{
		ID:     1,
		UserID: 100,
		Side:   SideBuy,
		Kind:   OrderKindLimit,
		Price:  100,
		Size:   10,
		Time:   1000000,
	}

	report, events, err := c.SubmitLimit(order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.OrderID != 1 {
		t.Errorf("expected OrderID 1, got %d", report.OrderID)
	}
	if report.Remaining != 10 {
		t.Errorf("expected remaining 10, got %d", report.Remaining)
	}
	if !report.Rested {
		t.Error("expected order to rest on book")
	}
	if len(report.Fills) != 0 {
		t.Errorf("expected no fills, got %d", len(report.Fills))
	}

	// Should have OrderRestedEvent
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if _, ok := events[0].(OrderRestedEvent); !ok {
		t.Errorf("expected OrderRestedEvent, got %T", events[0])
	}
}

func TestSubmitMarketAgainstLimitOrders(t *testing.T) {
	c := NewCore()

	// Submit a sell limit order
	sellOrder := Order{
		ID:     1,
		UserID: 100,
		Side:   SideSell,
		Kind:   OrderKindLimit,
		Price:  100,
		Size:   10,
		Time:   1000000,
	}
	_, _, err := c.SubmitLimit(sellOrder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Submit a buy market order
	buyOrder := Order{
		ID:     2,
		UserID: 200,
		Side:   SideBuy,
		Kind:   OrderKindMarket,
		Size:   5,
		Time:   2000000,
	}

	report, events, err := c.SubmitMarket(buyOrder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Remaining != 0 {
		t.Errorf("expected remaining 0, got %d", report.Remaining)
	}
	if len(report.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(report.Fills))
	}
	if report.Fills[0].Size != 5 {
		t.Errorf("expected fill size 5, got %d", report.Fills[0].Size)
	}

	// Should have TradeEvent and OrderReducedEvent
	tradeFound := false
	reducedFound := false
	for _, ev := range events {
		switch ev.(type) {
		case TradeEvent:
			tradeFound = true
		case OrderReducedEvent:
			reducedFound = true
		}
	}
	if !tradeFound {
		t.Error("expected TradeEvent")
	}
	if !reducedFound {
		t.Error("expected OrderReducedEvent")
	}
}

func TestCancel(t *testing.T) {
	c := NewCore()

	// Submit a buy limit order
	order := Order{
		ID:     1,
		UserID: 100,
		Side:   SideBuy,
		Kind:   OrderKindLimit,
		Price:  100,
		Size:   10,
		Time:   1000000,
	}
	_, _, err := c.SubmitLimit(order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cancel the order
	report, events, err := c.Cancel(1, 2000000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.OrderID != 1 {
		t.Errorf("expected OrderID 1, got %d", report.OrderID)
	}
	if report.CanceledSize != 10 {
		t.Errorf("expected canceled size 10, got %d", report.CanceledSize)
	}

	// Should have OrderRemovedEvent
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	removed, ok := events[0].(OrderRemovedEvent)
	if !ok {
		t.Fatalf("expected OrderRemovedEvent, got %T", events[0])
	}
	if removed.Reason != RemoveReasonCanceled {
		t.Errorf("expected reason Canceled, got %v", removed.Reason)
	}
}

func TestValidation(t *testing.T) {
	c := NewCore()

	tests := []struct {
		name  string
		order Order
	}{
		{"zero ID", Order{ID: 0, UserID: 1, Side: SideBuy, Kind: OrderKindLimit, Price: 100, Size: 10, Time: 1000}},
		{"zero UserID", Order{ID: 1, UserID: 0, Side: SideBuy, Kind: OrderKindLimit, Price: 100, Size: 10, Time: 1000}},
		{"zero Size", Order{ID: 1, UserID: 1, Side: SideBuy, Kind: OrderKindLimit, Price: 100, Size: 0, Time: 1000}},
		{"zero Price for limit", Order{ID: 1, UserID: 1, Side: SideBuy, Kind: OrderKindLimit, Price: 0, Size: 10, Time: 1000}},
		{"zero Time", Order{ID: 1, UserID: 1, Side: SideBuy, Kind: OrderKindLimit, Price: 100, Size: 10, Time: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.SubmitLimit(tt.order)
			if err != ErrInvalidOrder {
				t.Errorf("expected ErrInvalidOrder, got %v", err)
			}
		})
	}
}
