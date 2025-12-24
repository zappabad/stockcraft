# Architecture Overview

## System Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                   Game                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                          MarketService                               │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │    │
│  │  │ OrderbookSvc │  │ OrderbookSvc │  │ OrderbookSvc │  ...          │    │
│  │  │   (AAPL)     │  │   (GOOGL)    │  │   (MSFT)     │               │    │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘               │    │
│  │         │                 │                 │                        │    │
│  │         └─────────────────┴─────────────────┘                        │    │
│  │                           │                                          │    │
│  │                    MarketView (aggregated)                           │    │
│  │                           │                                          │    │
│  │                    Events() ──────────────────────────────┐          │    │
│  └───────────────────────────────────────────────────────────┼──────────┘    │
│                                                              │               │
│  ┌─────────────────┐    ┌─────────────────┐                  │               │
│  │   NewsService   │    │  TraderRunner   │◄─────────────────┤               │
│  │                 │    │   (Strategy)    │                  │               │
│  │  Events() ──────┼────┼─► Events() ─────┼──────────────────┤               │
│  └─────────────────┘    └─────────────────┘                  │               │
│                                                              │               │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                        BrokerService                                 │    │
│  │  AttachTraderEvents() ◄───────────────────────────────────┘          │    │
│  │  AttachNewsEvents()                                                  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                            ┌───────────────┐
                            │   cmd/tui     │
                            │  (Display)    │
                            └───────────────┘
```

## Data Flow

### Order Submission Flow

```
User/Trader                Market                 Orderbook              Core
    │                        │                       │                    │
    │  SubmitLimit()         │                       │                    │
    ├───────────────────────►│                       │                    │
    │                        │  SubmitLimit()        │                    │
    │                        ├──────────────────────►│                    │
    │                        │                       │  command           │
    │                        │                       ├───────────────────►│
    │                        │                       │                    │
    │                        │                       │  SubmitLimit()     │
    │                        │                       │◄───────────────────┤
    │                        │                       │  (report, events)  │
    │                        │                       │                    │
    │                        │                       │  emitEvent()       │
    │                        │                       ├────┐               │
    │                        │                       │    │ internal      │
    │                        │                       │◄───┘ channel       │
    │                        │  report               │                    │
    │  report                │◄──────────────────────┤                    │
    │◄───────────────────────┤                       │                    │
```

### Event Propagation Flow

```
Core Events                 Orderbook Service       Market Service        Subscribers
    │                            │                       │                    │
    │  []Event                   │                       │                    │
    ├───────────────────────────►│                       │                    │
    │                            │                       │                    │
    │                    ┌───────┴───────┐               │                    │
    │                    │               │               │                    │
    │              internalEvents   externalEvents       │                    │
    │                    │               │               │                    │
    │                    ▼               │               │                    │
    │              view.Apply()          │               │                    │
    │                                    │               │                    │
    │                                    ├──────────────►│                    │
    │                                    │  core.Event   │                    │
    │                                    │               │                    │
    │                                    │        mview.Apply()               │
    │                                    │               │                    │
    │                                    │               │  MarketEvent       │
    │                                    │               ├───────────────────►│
    │                                    │               │                    │
```

## Threading Model

### Goroutine Ownership

Each service owns specific goroutines:

| Service | Goroutines | Purpose |
|---------|------------|---------|
| `OrderbookService` | 2 | Command processor, Event dispatcher |
| `MarketService` | N | One event forwarder per ticker |
| `NewsService` | 1 | Event dispatcher |
| `TraderRunner` | 1 | Strategy tick loop |
| `BrokerService` | 2 | Trader event listener, News event listener |

### Channel Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    OrderbookService                          │
│                                                              │
│  cmdCh (buffered)                                            │
│    ├─► Command Processor ──► core.SubmitLimit/etc           │
│                                    │                         │
│                              []core.Event                    │
│                                    │                         │
│  internalEvents (buffered)  ◄──────┘                         │
│    │                                                         │
│    └─► Event Dispatcher                                      │
│           │                                                  │
│           ├─► view.Apply()  (authoritative, never drops)    │
│           │                                                  │
│           └─► externalEvents (may drop if configured)        │
│                    │                                         │
└────────────────────┼─────────────────────────────────────────┘
                     │
                     ▼
              External Subscribers
```

## Shutdown Sequence

The `Game.Close()` method shuts down subsystems in reverse dependency order:

```
1. Stop Traders      (no more orders generated)
2. Stop News         (no more news published)
3. Stop Market       (closes all orderbooks)
4. Stop Broker       (event listeners finish)
```

Each service uses:
- `chan struct{}` for close signal
- `sync.Once` for close idempotency
- `sync.WaitGroup` to wait for goroutine completion

## Key Design Decisions

### 1. Core Has No Side Effects

The `orderbook/core` package:
- Takes `Order.Time` as a parameter (doesn't call `time.Now()`)
- Takes `Order.ID` as a parameter (doesn't generate IDs)
- Has no channels, mutexes, or goroutines
- Returns `(Report, []Event, error)` synchronously

This makes the matching logic deterministic and easily testable.

### 2. Views Are Read Models

Views maintain a projection of state from events:
- Updated via `Apply(event)` method
- Thread-safe with `sync.RWMutex`
- Return copies, never internal references
- Can be rebuilt by replaying events

### 3. Services Own Concurrency

Services:
- Generate IDs (`atomic.Int64` counter)
- Timestamp orders (`time.Now().UnixNano()`)
- Own goroutines and channels
- Provide thread-safe public API

### 4. Events Are Values

All event types are value types (no pointers):
- Safe to pass across goroutines
- No aliasing issues
- Easy to serialize (future persistence)

### 5. External Events May Drop

External event channels can be configured to drop events if full:
- Prevents slow subscribers from blocking core processing
- View correctness is guaranteed by internal (non-dropping) channel
- Dropped count is tracked for monitoring
