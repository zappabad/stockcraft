# Orderbook System

The orderbook system is the foundation of Stockcraft, handling order matching and book state management.

## Package Structure

```
/internal/orderbook
  /core               # Deterministic matching engine
    types.go          # Side, OrderKind, PriceTicks, Size, OrderID, UserID, Order
    events.go         # Event interface and event types
    book.go           # Internal book data structures (heap, levels, linked list)
    core.go           # Core matching engine API

  /view               # Read-only state projection
    tape.go           # Ring buffer for trades
    view.go           # BookView with levels and orders

  /service            # Thread-safe service layer
    config.go         # Service configuration
    service.go        # Goroutine owner, public API
```

## Core Package (`/internal/orderbook/core`)

### Types

```go
type Side uint8
const (
    SideBuy Side = iota
    SideSell
)

type OrderKind uint8
const (
    OrderKindLimit OrderKind = iota
    OrderKindMarket
)

type PriceTicks int64    // Price in integer ticks
type Size int64          // Order quantity
type OrderID int64       // Unique order identifier
type UserID int64        // User/trader identifier

type Order struct {
    ID     OrderID
    UserID UserID
    Side   Side
    Kind   OrderKind
    Price  PriceTicks  // Limit orders only
    Size   Size
    Time   int64       // Unix nanos (set by service)
}
```

### Events

All events implement the `Event` interface:

```go
type Event interface {
    isEvent()  // marker method
}
```

| Event | Description | Fields |
|-------|-------------|--------|
| `TradeEvent` | A trade occurred | Price, Size, TakerSide, Time, TakerOrderID, TakerUserID, MakerOrderID, MakerUserID |
| `OrderRestedEvent` | Order placed on book | OrderID, UserID, Side, Price, Size, Time |
| `OrderReducedEvent` | Resting order partially filled | OrderID, Delta (negative), Remaining, Price, Side, UserID, MatchTime |
| `OrderRemovedEvent` | Order removed from book | OrderID, Reason, Remaining, Price, Side, UserID, Time |

### Core API

```go
type Core struct { ... }

func NewCore() *Core

func (c *Core) SubmitLimit(o Order) (SubmitReport, []Event, error)
func (c *Core) SubmitMarket(o Order) (SubmitReport, []Event, error)
func (c *Core) Cancel(id OrderID, now int64) (CancelReport, []Event, error)
```

**SubmitReport:**
```go
type SubmitReport struct {
    OrderID   OrderID
    Remaining Size      // Size left after matching
    Fills     []Fill    // Fills from this order
    Rested    bool      // Whether order rested on book
}

type Fill struct {
    MakerOrderID OrderID
    Price        PriceTicks
    Size         Size
}
```

**CancelReport:**
```go
type CancelReport struct {
    OrderID      OrderID
    CanceledSize Size
}
```

### Validation Rules

Orders are rejected (`ErrInvalidOrder`) if:
- `ID == 0` or `UserID == 0`
- `Size <= 0`
- `Price <= 0` (limit orders only)
- `Side` is not `SideBuy` or `SideSell`
- `Kind` doesn't match method (limit vs market)
- `Time <= 0`

Duplicate IDs return `ErrDuplicateID`.

### Matching Algorithm

1. **Price-Time Priority**: Orders match at the best available price, with earlier orders at same price matching first.

2. **Limit Order Matching**:
   - Buy limit matches asks at or below limit price
   - Sell limit matches bids at or above limit price
   - Unmatched remainder rests on book

3. **Market Order Matching**:
   - Matches against opposite side until filled
   - Never rests on book
   - May have remaining size if insufficient liquidity

### Internal Data Structures

```
┌─────────────────────────────────────────┐
│              orderBook                   │
│  ┌─────────────┐   ┌─────────────┐      │
│  │   bids      │   │   asks      │      │
│  │ (bookSide)  │   │ (bookSide)  │      │
│  └──────┬──────┘   └──────┬──────┘      │
│         │                 │              │
│         ▼                 ▼              │
│    ┌─────────┐       ┌─────────┐        │
│    │levelHeap│       │levelHeap│        │
│    │(max-heap│       │(min-heap│        │
│    │for bids)│       │for asks)│        │
│    └────┬────┘       └────┬────┘        │
│         │                 │              │
│         ▼                 ▼              │
│    levels map[PriceTicks]*level         │
│                                          │
│    orders map[OrderID]*restingOrder     │
└─────────────────────────────────────────┘

level:
  price: PriceTicks
  head ──► restingOrder ──► restingOrder ──► ... (doubly linked list)
  tail ◄──────────────────────────────────────────┘
  totalVolume: Size
```

## View Package (`/internal/orderbook/view`)

### TradeTape

Ring buffer for storing recent trades:

```go
type TradeTape struct { ... }

func NewTradeTape(capacity int) *TradeTape
func (t *TradeTape) Append(tr core.TradeEvent)
func (t *TradeTape) Last(n int) []core.TradeEvent  // Returns copies
```

### BookView

Read-only projection of book state:

```go
type BookView struct { ... }

func NewBookView(tapeCapacity int) *BookView
func (v *BookView) Apply(ev core.Event)

// Snapshot methods (return copies, never internal references)
func (v *BookView) Levels(side core.Side) []Level
func (v *BookView) Orders(side core.Side) []RestingOrder
func (v *BookView) TradesLast(n int) []core.TradeEvent
```

**Level:**
```go
type Level struct {
    Price core.PriceTicks
    Size  core.Size  // Aggregate size at this price
}
```

**RestingOrder:**
```go
type RestingOrder struct {
    ID     core.OrderID
    UserID core.UserID
    Side   core.Side
    Price  core.PriceTicks
    Size   core.Size
    Time   int64
}
```

### Thread Safety

`BookView` uses `sync.RWMutex`:
- `Apply()` takes write lock
- Snapshot methods take read lock
- All returns are deep copies

## Service Package (`/internal/orderbook/service`)

### Configuration

```go
type Config struct {
    CommandBuffer       int   // Inbound command channel size (default: 256)
    EventBuffer         int   // Internal event channel size (default: 1024)
    TradeTapeSize       int   // Trade history capacity (default: 1000)
    DropExternalEvents  bool  // Drop external events on overflow (default: true)
    ExternalEventBuffer int   // External event channel size (default: 256)
}
```

### Service API

```go
type Service struct { ... }

func NewService(cfg Config) *Service

// Order operations (thread-safe, blocking)
func (s *Service) SubmitLimit(ctx, userID, side, price, size) (SubmitReport, error)
func (s *Service) SubmitMarket(ctx, userID, side, size) (SubmitReport, error)
func (s *Service) Cancel(ctx, orderID) (CancelReport, error)

// View access (read-only, thread-safe)
func (s *Service) GetLevels(side) []view.Level
func (s *Service) GetOrders(side) []view.RestingOrder
func (s *Service) GetTradesLast(n) []core.TradeEvent

// Event subscription
func (s *Service) Events() <-chan core.Event

// Lifecycle
func (s *Service) Close()
func (s *Service) DroppedExternalEvents() int64
```

### Internal Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Service                              │
│                                                              │
│  Goroutine 1: Command Processor                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  for {                                               │    │
│  │    select {                                          │    │
│  │    case <-closed: return                            │    │
│  │    case cmd := <-cmdCh:                             │    │
│  │      processCommand(cmd)                            │    │
│  │    }                                                 │    │
│  │  }                                                   │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  Goroutine 2: Event Dispatcher                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  for {                                               │    │
│  │    select {                                          │    │
│  │    case <-closed: return                            │    │
│  │    case ev := <-internalEvents:                     │    │
│  │      view.Apply(ev)           // Always             │    │
│  │      externalEvents <- ev     // May drop           │    │
│  │    }                                                 │    │
│  │  }                                                   │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  State:                                                      │
│    core *core.Core           // Matching engine             │
│    view *view.BookView       // Read model                  │
│    idGen atomic.Int64        // ID generator                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### ID Generation

- Service generates OrderIDs using `atomic.Int64`
- Initialized from `time.Now().UnixNano()` to avoid collisions across restarts
- Each order gets `idGen.Add(1)`

### Timestamping

- Service sets `Order.Time = time.Now().UnixNano()` before passing to core
- Core never calls `time.Now()` - it's deterministic

## Usage Example

```go
// Create service
cfg := service.DefaultConfig()
svc := service.NewService(cfg)
defer svc.Close()

// Subscribe to events
go func() {
    for ev := range svc.Events() {
        switch e := ev.(type) {
        case core.TradeEvent:
            fmt.Printf("Trade: %d @ %d\n", e.Size, e.Price)
        }
    }
}()

// Submit orders
ctx := context.Background()
report, err := svc.SubmitLimit(ctx, userID, core.SideBuy, 100, 10)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Order %d placed, remaining: %d\n", report.OrderID, report.Remaining)

// Query book state
levels := svc.GetLevels(core.SideBuy)
for _, lvl := range levels {
    fmt.Printf("  %d @ %d\n", lvl.Size, lvl.Price)
}
```
