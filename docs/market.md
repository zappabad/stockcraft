# Market System

The market system aggregates multiple order books (one per ticker) and provides a unified interface for multi-asset trading.

## Package Structure

```
/internal/market
  types.go              # TickerID, MarketSnapshot
  /view
    events.go           # MarketEvent interface
    view.go             # Read-only market projection
  /service
    config.go           # Service configuration
    service.go          # Multi-orderbook manager
```

## Types

```go
type TickerID string

type MarketSnapshot struct {
    BestBid   int64  // Best bid price (0 if no bids)
    BestAsk   int64  // Best ask price (0 if no asks)
    BidVolume int64  // Total bid volume at best bid
    AskVolume int64  // Total ask volume at best ask
    LastPrice int64  // Last trade price (0 if no trades)
    LastSize  int64  // Last trade size
}
```

## View Package (`/internal/market/view`)

### Events

```go
type MarketEvent interface {
    isMarketEvent()
    Ticker() TickerID
}

type OrderBookEvent struct {
    TickerID TickerID
    Event    core.Event  // Underlying orderbook event
}
```

### MarketView

```go
type MarketView struct { ... }

func NewMarketView() *MarketView
func (v *MarketView) Apply(ev MarketEvent)

// Snapshots (thread-safe, returns copies)
func (v *MarketView) Snapshot(ticker TickerID) MarketSnapshot
func (v *MarketView) AllSnapshots() map[TickerID]MarketSnapshot
func (v *MarketView) Tickers() []TickerID
```

**Thread Safety:**
- Uses `sync.RWMutex`
- `Apply()` takes write lock
- Snapshot methods take read lock

## Service Package (`/internal/market/service`)

### Configuration

```go
type Config struct {
    Tickers             []TickerID  // List of tickers to manage
    OrderBookConfig     observice.Config  // Config for each orderbook
    EventBuffer         int         // Unified event channel size (default: 1024)
    DropEvents          bool        // Drop events on overflow (default: true)
}
```

### MarketService

```go
type MarketService struct { ... }

func NewMarketService(cfg Config) *MarketService

// Order operations (routed to appropriate orderbook)
func (s *MarketService) SubmitLimit(ctx, ticker, userID, side, price, size) (SubmitReport, error)
func (s *MarketService) SubmitMarket(ctx, ticker, userID, side, size) (SubmitReport, error)
func (s *MarketService) Cancel(ctx, ticker, orderID) (CancelReport, error)

// View access
func (s *MarketService) Snapshot(ticker TickerID) MarketSnapshot
func (s *MarketService) AllSnapshots() map[TickerID]MarketSnapshot
func (s *MarketService) GetLevels(ticker, side) []view.Level

// Access underlying orderbook for a specific ticker
func (s *MarketService) OrderBook(ticker TickerID) *observice.Service

// Events (unified stream from all orderbooks)
func (s *MarketService) Events() <-chan view.MarketEvent

// Lifecycle
func (s *MarketService) Close()
```

### Internal Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        MarketService                                 │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    Orderbook Map                               │  │
│  │  map[TickerID]*observice.Service                              │  │
│  │                                                                │  │
│  │    AAPL ──► orderbook.Service ──► events ──┐                 │  │
│  │    GOOG ──► orderbook.Service ──► events ──┼──► fanIn       │  │
│  │    MSFT ──► orderbook.Service ──► events ──┘     goroutine   │  │
│  │                                                      │        │  │
│  └───────────────────────────────────────────────────────┼────────┘  │
│                                                          │           │
│  ┌───────────────────────────────────────────────────────▼────────┐  │
│  │                   Fan-in Goroutine                             │  │
│  │  for each ticker {                                             │  │
│  │    go func(t TickerID, ch <-chan core.Event) {                │  │
│  │      for ev := range ch {                                      │  │
│  │        wrap := OrderBookEvent{t, ev}                          │  │
│  │        view.Apply(wrap)                                        │  │
│  │        select {                                                │  │
│  │          case events <- wrap: // ok                           │  │
│  │          default: dropped++   // overflow                     │  │
│  │        }                                                       │  │
│  │      }                                                         │  │
│  │    }(ticker, orderbook.Events())                              │  │
│  │  }                                                             │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                       MarketView                               │  │
│  │  Per-ticker snapshots:                                         │  │
│  │    - BestBid/BestAsk                                          │  │
│  │    - Volumes                                                   │  │
│  │    - Last trade price/size                                     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Event Flow

1. Orders come in via `SubmitLimit`/`SubmitMarket`/`Cancel`
2. Routed to appropriate `orderbook.Service` by ticker
3. Each orderbook publishes events on its channel
4. Fan-in goroutines wrap events with ticker info
5. `MarketView.Apply()` updates aggregate state
6. Events published to unified `Events()` channel

### Snapshot Updates

The `MarketView` updates snapshots based on orderbook events:

| Event | Update |
|-------|--------|
| `TradeEvent` | Update `LastPrice`, `LastSize` |
| `OrderRestedEvent` | Recalculate best bid/ask |
| `OrderReducedEvent` | Recalculate volumes |
| `OrderRemovedEvent` | Recalculate best bid/ask |

## Usage Example

```go
// Create market service with 3 tickers
cfg := service.Config{
    Tickers: []TickerID{"AAPL", "GOOG", "MSFT"},
    OrderBookConfig: observice.DefaultConfig(),
    EventBuffer: 1024,
}
market := service.NewMarketService(cfg)
defer market.Close()

// Subscribe to all market events
go func() {
    for ev := range market.Events() {
        ticker := ev.Ticker()
        switch e := ev.(type) {
        case view.OrderBookEvent:
            if trade, ok := e.Event.(core.TradeEvent); ok {
                fmt.Printf("[%s] Trade: %d @ %d\n", ticker, trade.Size, trade.Price)
            }
        }
    }
}()

// Submit order to specific ticker
ctx := context.Background()
report, err := market.SubmitLimit(ctx, "AAPL", userID, core.SideBuy, 150_00, 100)

// Get market snapshot
snap := market.Snapshot("AAPL")
fmt.Printf("AAPL: Bid %d x %d | Ask %d x %d\n",
    snap.BestBid, snap.BidVolume, snap.BestAsk, snap.AskVolume)

// Get all snapshots
for ticker, snap := range market.AllSnapshots() {
    fmt.Printf("%s: Last %d\n", ticker, snap.LastPrice)
}
```

## Design Decisions

### Why Per-Ticker Orderbooks?

Each ticker has its own `orderbook.Service` instance:
- **Isolation**: One orderbook can't block another
- **Scalability**: Easy to add/remove tickers
- **Parallelism**: Each orderbook runs its own goroutines

### Why Fan-In Events?

The unified event channel simplifies consumption:
- Single subscription point for all market activity
- Events tagged with ticker for filtering
- No need to manage multiple subscriptions

### Thread Safety Model

```
Writes:                         Reads:
  SubmitLimit()                   Snapshot()
  SubmitMarket()                  AllSnapshots()
  Cancel()                        GetLevels()
       │                               │
       ▼                               ▼
  ┌─────────────┐              ┌─────────────┐
  │ cmd channel │              │  RWMutex    │
  │ (buffered)  │              │  .RLock()   │
  └─────────────┘              └─────────────┘
       │                               │
       ▼                               ▼
  orderbook.Service            MarketView
  processes sequentially       returns copies
```

Write operations are serialized through command channels.
Read operations use shared locks and return copies.
This provides safe concurrent access without blocking readers.
