# Trader System

The trader system provides an interface for algorithmic trading strategies and a runner that executes them on a configurable tick schedule.

## Package Structure

```
/internal/trader
  types.go              # TradeDecision, Fill types
  /strategy
    interface.go        # Strategy interface
    example_strategy.go # Simple example implementation
  /runner
    config.go           # Runner configuration
    runner.go           # Tick-based strategy executor
```

## Types

```go
type TradeDecision struct {
    Ticker   TickerID
    Side     Side
    Kind     OrderKind
    Price    int64  // For limit orders
    Size     int64
}

type Fill struct {
    Ticker   TickerID
    OrderID  int64
    Price    int64
    Size     int64
    Side     Side
}
```

## Strategy Package (`/internal/trader/strategy`)

### Strategy Interface

```go
type Strategy interface {
    // Name returns the strategy's identifier
    Name() string
    
    // OnTick is called on each tick interval
    // Returns decisions to execute
    OnTick(ctx StrategyContext) []TradeDecision
    
    // OnFill is called when an order is filled
    OnFill(fill Fill)
}

type StrategyContext struct {
    Snapshots map[TickerID]MarketSnapshot  // Current market state
    Portfolio map[TickerID]int64           // Current positions
    Cash      int64                        // Available cash
    Time      int64                        // Current time (unix nanos)
}
```

### Example Strategy

A simple market maker that posts bid/ask quotes:

```go
type ExampleStrategy struct {
    spread   int64
    size     int64
    tickers  []TickerID
}

func NewExampleStrategy(tickers []TickerID, spread, size int64) *ExampleStrategy

func (s *ExampleStrategy) Name() string { return "example-market-maker" }

func (s *ExampleStrategy) OnTick(ctx StrategyContext) []TradeDecision {
    var decisions []TradeDecision
    
    for _, ticker := range s.tickers {
        snap := ctx.Snapshots[ticker]
        mid := (snap.BestBid + snap.BestAsk) / 2
        if mid == 0 {
            continue
        }
        
        decisions = append(decisions,
            TradeDecision{
                Ticker: ticker,
                Side:   SideBuy,
                Kind:   OrderKindLimit,
                Price:  mid - s.spread/2,
                Size:   s.size,
            },
            TradeDecision{
                Ticker: ticker,
                Side:   SideSell,
                Kind:   OrderKindLimit,
                Price:  mid + s.spread/2,
                Size:   s.size,
            },
        )
    }
    
    return decisions
}

func (s *ExampleStrategy) OnFill(fill Fill) {
    // Track fills for P&L calculation
}
```

## Runner Package (`/internal/trader/runner`)

### Configuration

```go
type Config struct {
    TickInterval  time.Duration  // How often to call OnTick (default: 100ms)
    UserID        int64          // UserID for order submission
    InitialCash   int64          // Starting cash balance
}
```

### TraderRunner

```go
type TraderRunner struct { ... }

func NewTraderRunner(
    cfg Config,
    strategy Strategy,
    market *MarketService,
) *TraderRunner

// Lifecycle
func (r *TraderRunner) Start()
func (r *TraderRunner) Stop()

// State
func (r *TraderRunner) Portfolio() map[TickerID]int64
func (r *TraderRunner) Cash() int64
func (r *TraderRunner) PnL() int64
```

### Internal Architecture

```
┌───────────────────────────────────────────────────────────────────┐
│                         TraderRunner                               │
│                                                                    │
│  Start()                                                           │
│     │                                                              │
│     ▼                                                              │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                   Tick Goroutine                             │  │
│  │  ticker := time.NewTicker(cfg.TickInterval)                 │  │
│  │  for {                                                       │  │
│  │    select {                                                  │  │
│  │    case <-stopCh: return                                    │  │
│  │    case <-ticker.C:                                         │  │
│  │      ctx := buildContext()                                  │  │
│  │      decisions := strategy.OnTick(ctx)                      │  │
│  │      executeDecisions(decisions)                            │  │
│  │    }                                                         │  │
│  │  }                                                           │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                    │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                  Fill Listener Goroutine                     │  │
│  │  for ev := range market.Events() {                          │  │
│  │    if trade, ok := ev.(TradeEvent); ok {                    │  │
│  │      if trade.TakerUserID == cfg.UserID {                   │  │
│  │        updatePosition(trade)                                 │  │
│  │        strategy.OnFill(toFill(trade))                       │  │
│  │      }                                                       │  │
│  │    }                                                         │  │
│  │  }                                                           │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                    │
│  State (protected by sync.RWMutex):                               │
│    portfolio map[TickerID]int64  // Position per ticker           │
│    cash      int64               // Available cash                │
│                                                                    │
└───────────────────────────────────────────────────────────────────┘
```

### Execution Flow

```
┌──────────┐     OnTick()      ┌──────────────┐
│ Strategy │ ──────────────►  │ []Decision   │
└──────────┘                   └──────┬───────┘
                                      │
                                      ▼
                              ┌──────────────┐
                              │ For each:    │
                              │  - Validate  │
                              │  - Submit    │
                              └──────┬───────┘
                                      │
                  ┌───────────────────┴───────────────────┐
                  │                                       │
                  ▼                                       ▼
           ┌────────────┐                         ┌────────────┐
           │ Limit      │                         │ Market     │
           │ market.    │                         │ market.    │
           │ SubmitLimit│                         │ SubmitMarket│
           └────────────┘                         └────────────┘
                  │                                       │
                  └───────────────────┬───────────────────┘
                                      │
                                      ▼
                              ┌──────────────┐
                              │ Fill events  │
                              │ from market  │
                              └──────┬───────┘
                                      │
                                      ▼
                              ┌──────────────┐
                              │ Update       │
                              │ portfolio,   │
                              │ cash, call   │
                              │ OnFill()     │
                              └──────────────┘
```

### Position Tracking

The runner tracks positions from fill events:

```go
func (r *TraderRunner) onTrade(trade TradeEvent) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if trade.TakerUserID != r.cfg.UserID {
        return  // Not our trade
    }
    
    ticker := trade.Ticker  // From MarketEvent wrapper
    
    if trade.TakerSide == SideBuy {
        r.portfolio[ticker] += int64(trade.Size)
        r.cash -= int64(trade.Price) * int64(trade.Size)
    } else {
        r.portfolio[ticker] -= int64(trade.Size)
        r.cash += int64(trade.Price) * int64(trade.Size)
    }
    
    r.strategy.OnFill(Fill{
        Ticker:  ticker,
        OrderID: trade.TakerOrderID,
        Price:   int64(trade.Price),
        Size:    int64(trade.Size),
        Side:    trade.TakerSide,
    })
}
```

## Writing a Strategy

### Basic Template

```go
type MyStrategy struct {
    // Strategy parameters
    threshold float64
    maxPos    int64
}

func (s *MyStrategy) Name() string {
    return "my-strategy"
}

func (s *MyStrategy) OnTick(ctx StrategyContext) []TradeDecision {
    var decisions []TradeDecision
    
    for ticker, snap := range ctx.Snapshots {
        pos := ctx.Portfolio[ticker]
        
        // Your logic here
        // Example: momentum strategy
        if snap.LastPrice > snap.BestAsk && pos < s.maxPos {
            decisions = append(decisions, TradeDecision{
                Ticker: ticker,
                Side:   SideBuy,
                Kind:   OrderKindMarket,
                Size:   100,
            })
        }
    }
    
    return decisions
}

func (s *MyStrategy) OnFill(fill Fill) {
    // Log fills, update internal state, etc.
    log.Printf("Filled: %s %d @ %d", fill.Ticker, fill.Size, fill.Price)
}
```

### Best Practices

1. **Idempotency**: `OnTick` may be called multiple times at the same logical time
2. **No Blocking**: Don't block in `OnTick` - the runner has a timeout
3. **State Management**: Use `OnFill` to track actual fills, not submissions
4. **Position Limits**: Check `ctx.Portfolio` before making decisions
5. **Error Handling**: The runner logs but doesn't crash on strategy errors

### Available Context

Each `OnTick` receives:

| Field | Type | Description |
|-------|------|-------------|
| `Snapshots` | `map[TickerID]MarketSnapshot` | Current best bid/ask, last trade |
| `Portfolio` | `map[TickerID]int64` | Current position per ticker |
| `Cash` | `int64` | Available cash (after fills) |
| `Time` | `int64` | Current unix nanoseconds |

## Usage Example

```go
// Create market service
market := marketservice.NewMarketService(marketCfg)

// Create strategy
strat := strategy.NewExampleStrategy(
    []TickerID{"AAPL", "GOOG"},
    10,   // spread
    100,  // size per quote
)

// Create runner
runnerCfg := runner.Config{
    TickInterval: 100 * time.Millisecond,
    UserID:       42,
    InitialCash:  1_000_000,
}
trader := runner.NewTraderRunner(runnerCfg, strat, market)

// Start trading
trader.Start()

// ... let it run ...

// Stop and get results
trader.Stop()
fmt.Printf("Final P&L: %d\n", trader.PnL())
fmt.Printf("Positions: %v\n", trader.Portfolio())
```

## Design Decisions

### Why Tick-Based?

- **Simplicity**: Strategies don't manage their own timing
- **Determinism**: Easier to test with controlled tick calls
- **Fairness**: All strategies run at same interval
- **Resource Control**: Prevents runaway strategies

### Why Separate Fill Notifications?

Fills may happen asynchronously (e.g., limit order filled later):
- `OnTick` returns decisions
- `OnFill` receives actual fills
- Position tracking is event-driven, not assumed from submissions

### Thread Safety

- Runner state protected by `sync.RWMutex`
- Strategy calls are serialized (one `OnTick` at a time)
- `OnFill` may be called from different goroutine than `OnTick`
