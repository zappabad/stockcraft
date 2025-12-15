package engine

import (
	"testing"
)

func mustSubmitLimit(t *testing.T, ob *OrderBook, o *Order) []Match {
	t.Helper()
	matches, _, err := ob.SubmitLimitOrder(o)
	if err != nil {
		t.Fatalf("SubmitLimitOrder err=%v", err)
	}
	return matches
}

func mustSubmitMarket(t *testing.T, ob *OrderBook, o *Order) []Match {
	t.Helper()
	matches, err := ob.SubmitMarketOrder(o)
	if err != nil {
		t.Fatalf("SubmitMarketOrder err=%v", err)
	}
	return matches
}

func TestLimitOrder_RestsWhenNotCrossing(t *testing.T) {
	ob := NewOrderBook()

	mustSubmitLimit(t, ob, NewLimitOrder(1, 10, SideBuy, 100, 5))

	bp, bs, ok := ob.BestBid()
	if !ok || bp != 100 || bs != 5 {
		t.Fatalf("best bid got ok=%v price=%v size=%v", ok, bp, bs)
	}
	_, _, ok = ob.BestAsk()
	if ok {
		t.Fatalf("expected no asks")
	}
}

func TestLimitOrder_Crossing_BuyMatchesAskUpToLimit(t *testing.T) {
	ob := NewOrderBook()

	// Rest an ask at 101
	mustSubmitLimit(t, ob, NewLimitOrder(1, 10, SideSell, 101, 3))

	// Buy limit 102 should cross 101
	matches := mustSubmitLimit(t, ob, NewLimitOrder(2, 20, SideBuy, 102, 2))
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Price != 101 || matches[0].SizeFilled != 2 {
		t.Fatalf("match=%+v", matches[0])
	}

	// Remaining ask should be size 1 at 101
	ap, as, ok := ob.BestAsk()
	if !ok || ap != 101 || as != 1 {
		t.Fatalf("best ask got ok=%v price=%v size=%v", ok, ap, as)
	}
}

func TestLimitOrder_Crossing_DoesNotMatchBeyondLimit(t *testing.T) {
	ob := NewOrderBook()

	mustSubmitLimit(t, ob, NewLimitOrder(1, 10, SideSell, PriceTicks(101), Size(1)))
	mustSubmitLimit(t, ob, NewLimitOrder(2, 10, SideSell, PriceTicks(103), Size(1)))

	// Buy limit at 102 should match 101 but stop before 103.
	matches := mustSubmitLimit(t, ob, NewLimitOrder(3, 20, SideBuy, PriceTicks(102), Size(5)))
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Price != 101 || matches[0].SizeFilled != 1 {
		t.Fatalf("match=%+v", matches[0])
	}

	// Remaining buy should rest at 102 with size 4
	bp, bs, ok := ob.BestBid()
	if !ok || bp != 102 || bs != 4 {
		t.Fatalf("best bid got ok=%v price=%v size=%v", ok, bp, bs)
	}

	// Ask at 103 should still exist
	ap, as, ok := ob.BestAsk()
	if !ok || ap != 103 || as != 1 {
		t.Fatalf("best ask got ok=%v price=%v size=%v", ok, ap, as)
	}
}

func TestPriceTimePriority_FIFOWithinLevel(t *testing.T) {
	ob := NewOrderBook()

	// Two asks at same price; first should fill first.
	ask1 := NewLimitOrder(1, 10, SideSell, 100, 2)
	ask2 := NewLimitOrder(2, 11, SideSell, 100, 2)
	mustSubmitLimit(t, ob, ask1)
	mustSubmitLimit(t, ob, ask2)

	// Buy market for 3 should consume ask1 fully (2) then 1 from ask2.
	matches := mustSubmitMarket(t, ob, NewMarketOrder(3, 20, SideBuy, 3))

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].Ask.ID != 1 || matches[0].SizeFilled != 2 {
		t.Fatalf("first match wrong: %+v", matches[0])
	}
	if matches[1].Ask.ID != 2 || matches[1].SizeFilled != 1 {
		t.Fatalf("second match wrong: %+v", matches[1])
	}

	// Remaining ask2 should be size 1
	ap, as, ok := ob.BestAsk()
	if !ok || ap != 100 || as != 1 {
		t.Fatalf("best ask got ok=%v price=%v size=%v", ok, ap, as)
	}
}

func TestPricePriority_BetterPriceWins(t *testing.T) {
	ob := NewOrderBook()

	// Two asks, different prices
	mustSubmitLimit(t, ob, NewLimitOrder(1, 10, SideSell, 99, 1))
	mustSubmitLimit(t, ob, NewLimitOrder(2, 10, SideSell, 100, 1))

	matches := mustSubmitMarket(t, ob, NewMarketOrder(3, 20, SideBuy, 1))
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Price != 99 {
		t.Fatalf("expected price 99, got %v", matches[0].Price)
	}
}

func TestMarketOrder_NeverRests(t *testing.T) {
	ob := NewOrderBook()

	// Empty book, market buy can't fill and should not rest.
	mustSubmitMarket(t, ob, NewMarketOrder(1, 10, SideBuy, 5))

	_, _, ok := ob.BestBid()
	if ok {
		t.Fatalf("expected no bids after unfilled market order")
	}
}

func TestCancelOrder_RemovesAndUpdatesLevelVolume(t *testing.T) {
	ob := NewOrderBook()

	mustSubmitLimit(t, ob, NewLimitOrder(1, 10, SideBuy, 100, 3))
	mustSubmitLimit(t, ob, NewLimitOrder(2, 11, SideBuy, 100, 2))

	// total at 100 is 5
	bp, bs, ok := ob.BestBid()
	if !ok || bp != 100 || bs != 5 {
		t.Fatalf("best bid got ok=%v price=%v size=%v", ok, bp, bs)
	}

	if !ob.CancelOrder(1) {
		t.Fatalf("expected cancel true")
	}

	// remaining total at 100 is 2
	bp, bs, ok = ob.BestBid()
	if !ok || bp != 100 || bs != 2 {
		t.Fatalf("best bid got ok=%v price=%v size=%v", ok, bp, bs)
	}

	if ob.CancelOrder(1) {
		t.Fatalf("expected cancel false for already canceled")
	}
}

func TestDuplicateOrderIDRejected(t *testing.T) {
	ob := NewOrderBook()

	_, _, err := ob.SubmitLimitOrder(NewLimitOrder(1, 10, SideBuy, 100, 1))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	_, _, err = ob.SubmitLimitOrder(NewLimitOrder(1, 11, SideBuy, 100, 1))
	if err == nil {
		t.Fatalf("expected duplicate ID error")
	}
}

func TestTradesRecorded(t *testing.T) {
	ob := NewOrderBook()

	mustSubmitLimit(t, ob, NewLimitOrder(1, 10, SideSell, 100, 2))
	mustSubmitMarket(t, ob, NewMarketOrder(2, 20, SideBuy, 1))

	trades := ob.Trades()
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Price != 100 || trades[0].Size != 1 || trades[0].TakerSide != SideBuy {
		t.Fatalf("trade=%+v", trades[0])
	}
}

func TestSnapshots(t *testing.T) {
	ob := NewOrderBook()

	mustSubmitLimit(t, ob, NewLimitOrder(1, 10, SideBuy, 100, 1))
	mustSubmitLimit(t, ob, NewLimitOrder(2, 10, SideBuy, 101, 2))
	mustSubmitLimit(t, ob, NewLimitOrder(3, 10, SideSell, 103, 3))
	mustSubmitLimit(t, ob, NewLimitOrder(4, 10, SideSell, 102, 4))

	bids := ob.BidsSnapshot()
	if len(bids) != 2 || bids[0].Price != 101 || bids[0].Size != 2 || bids[1].Price != 100 || bids[1].Size != 1 {
		t.Fatalf("bids snapshot=%+v", bids)
	}

	asks := ob.AsksSnapshot()
	if len(asks) != 2 || asks[0].Price != 102 || asks[0].Size != 4 || asks[1].Price != 103 || asks[1].Size != 3 {
		t.Fatalf("asks snapshot=%+v", asks)
	}
}
