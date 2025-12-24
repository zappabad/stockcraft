## Implementation specification

### 0) Go module + naming
- Assume `go.mod` module path is `MODULE_PATH` (agent must read `go.mod` and use it for imports).
- All code lives under `/internal/**` except binaries under `/cmd/**`.
- No package named `api`. Use “service” for goroutine owners.

---

## 1) Target folder/package structure

Create/move code into:

```
/internal
  /orderbook
    /core
      types.go
      events.go
      book.go
      core.go
    /view
      tape.go
      view.go
    /service
      config.go
      service.go

  /market
    types.go
    /view
      view.go
      events.go
    /service
      config.go
      service.go

  /news
    types.go
    /view
      view.go
      events.go
    /service
      config.go
      service.go

  /trader
    types.go
    /strategy
      interface.go
      example_strategy.go
    /runner
      config.go
      runner.go

  /broker
    types.go
    /service
      config.go
      service.go
    /view
      view.go

  /game
    config.go
    game.go

/cmd
  /tui
    main.go
```

Rules:
- `/internal/orderbook/core` must not import anything from `/internal/orderbook/service`, `/internal/market`, `/internal/news`, `/internal/trader`, `/internal/broker`, `/internal/game`, or `/cmd`.
- `/internal/orderbook/view` can import `/internal/orderbook/core` types/events only.
- `/internal/orderbook/service` can import `/internal/orderbook/core` and `/internal/orderbook/view` only.
- `/internal/market/*` can import `/internal/orderbook/service`, `/internal/orderbook/core` types/events, and `/internal/orderbook/view` snapshot types.
- Traders/broker/news/game may import market/news services and their snapshot types, but must not import orderbook/core directly unless only for shared primitive types (Side/PriceTicks/Size) and events.

---

## 2) Orderbook subsystem (core/view/service)

### 2.1 `/internal/orderbook/core` (deterministic)
Implement:
- `types.go`:
  - `Side`, `OrderKind`, `PriceTicks`, `Size`, `OrderID`, `UserID`, `Order` (value type)
- `events.go`:
  - event interface: `type Event interface { isEvent() }`
  - events: `TradeEvent`, `OrderRestedEvent`, `OrderReducedEvent`, `OrderRemovedEvent`
  - `RemoveReason` enum: `Filled`, `Canceled`
- `book.go` + `core.go`:
  - `Core` struct owns internal book state, resting orders stored internally (no pointers exposed).
  - Methods:
    - `SubmitLimit(o Order) (SubmitReport, []Event, error)`
    - `SubmitMarket(o Order) (SubmitReport, []Event, error)`
    - `Cancel(id OrderID, now int64) (CancelReport, []Event, error)`
  - Validation at boundary:
    - reject `ID==0`, `UserID==0`, `Size<=0`, `Price<=0` for limit, invalid side/kind, `Time<=0`.
  - No `time.Now()` usage inside `core`.
  - No goroutines, no mutexes, no channels.

Output structs:
- `SubmitReport{ OrderID, Remaining Size, Rested bool, Fills []Fill }`
- `Fill{ MakerOrderID, Price, Size }`
- `CancelReport{ OrderID, CanceledSize }`

### 2.2 `/internal/orderbook/view` (read model)
Implement:
- `TradeTape` ring buffer storing `core.TradeEvent` values (bounded).
- `BookView`:
  - Maintains:
    - `orders map[OrderID]orderState`
    - `bids map[PriceTicks]Size` totals
    - `asks map[PriceTicks]Size` totals
    - `TradeTape`
  - Method: `Apply(ev core.Event)`
  - Snapshot methods:
    - `Levels(side core.Side) []Level` (sorted best->worst)
    - `Orders(side core.Side) []RestingOrder` (sorted by price then time then id)
    - `TradesLast(n int) []core.TradeEvent`
- Snapshot structs:
  - `Level{ Price, Size }`
  - `RestingOrder{ ID, UserID, Side, Price, Size, Time }`

Threading:
- `BookView` uses `sync.RWMutex`. Snapshot returns must be copies (values), never internal references.

### 2.3 `/internal/orderbook/service` (goroutine owner + RPC)
Implement `Service` that:
- Owns `core.Core` and `view.BookView`.
- Has:
  - inbound command channel (buffered)
  - internal authoritative event channel (core->dispatcher) (buffered)
  - external events channel (buffered; droppable optional)
- Public methods:
  - `SubmitLimit(ctx, userID, side, price, size) (core.SubmitReport, error)`
  - `SubmitMarket(ctx, userID, side, size) (core.SubmitReport, error)`
  - `Cancel(ctx, id) (core.CancelReport, error)`
  - View-based getters (no core access):
    - `GetLevels(side) []view.Level`
    - `GetOrders(side) []view.RestingOrder`
    - `GetTradesLast(n) []core.TradeEvent`
  - `Events() <-chan core.Event`
  - `Close()`
  - `DroppedExternalEvents() int64`
- ID generation:
  - monotonic `atomic.Int64` counter; initialize from `time.Now().UnixNano()` in service constructor.
  - OrderID is generated in service, not core.
- Time stamping:
  - set `Order.Time = time.Now().UnixNano()` at submit; pass into core unchanged.

Config:
- `Config{ CommandBuffer, EventBuffer, TradeTapeSize, DropExternalEvents, ExternalEventBuffer }` with defaults.

Invariant:
- View correctness must rely on internal authoritative bus only (never droppable).
- External event dropping must not affect view.

---

## 3) Market subsystem (multi-book aggregation)

### 3.1 `/internal/market/types.go`
Define:
- `type TickerID int64`
- `type Ticker struct { ID int64; Name string; Decimals int8 }`
- Helper:
  - `func (t Ticker) TickerID() TickerID`

Do not use `Ticker` struct as map key.

### 3.2 `/internal/market/view`
Implement:
- `MarketEvent{ Ticker TickerID; Event core.Event }`
- `BestPrices{ BidPrice, BidSize, BidOK, AskPrice, AskSize, AskOK, LastPrice, LastTime, HasLast }`
- `MarketSnapshot{ ByTicker map[TickerID]BestPrices }`
- `MarketView`:
  - `Apply(tid TickerID, ev core.Event, book *orderbookservice.Service)`
  - `Snapshot() MarketSnapshot`

Policy:
- `Apply()` updates last trade from `TradeEvent`.
- For best bid/ask, compute by calling `book.GetLevels()` and taking `[0]` for each side (acceptable for TUI scale).
  - If later optimizing, compute incrementally; not required now.

Threading:
- `MarketView` uses `sync.RWMutex`. `Snapshot()` returns deep copies of maps.

### 3.3 `/internal/market/service`
Implement `MarketService` that:
- Owns:
  - `tickers map[TickerID]Ticker`
  - `books map[TickerID]*orderbookservice.Service`
  - `mview *marketview.MarketView`
  - consolidated external event channel `chan marketview.MarketEvent`
- Constructor:
  - `NewMarketService(tickers []Ticker, cfg Config) *MarketService`
  - Creates one orderbook service per ticker using embedded `orderbookservice.Config`.
  - Starts one goroutine per book that reads `book.Events()` and:
    - calls `mview.Apply(tid, ev, book)`
    - emits `MarketEvent` to external channel (droppable configurable)
- Public methods:
  - `SubmitLimit(ctx, tid, userID, side, price, size)`
  - `SubmitMarket(ctx, tid, userID, side, size)`
  - `Cancel(ctx, tid, orderID)`
  - Per-book view getters:
    - `GetLevels(tid, side)`
    - `GetOrders(tid, side)`
    - `GetTradesLast(tid, n)`
  - Market-level:
    - `Snapshot() MarketSnapshot`
    - `Events() <-chan MarketEvent`
  - `Close()`: closes market, closes all books, waits, closes external events channel.
- Errors:
  - `ErrUnknownTicker`

Config:
- `Config{ Book orderbookservice.Config; MarketEventBuffer int; DropMarketEvents bool }`

---

## 4) News subsystem (service/view; minimal)

### 4.1 `/internal/news/types.go`
Define:
- `type NewsID int64`
- `type NewsItem struct { ID NewsID; Time int64; Ticker market.TickerID (optional); Headline string; Body string; Severity int }`

### 4.2 `/internal/news/view`
- `NewsEvent{ Item NewsItem }`
- `NewsView` maintains ring buffer of `NewsItem` (bounded).
- Methods:
  - `Apply(NewsEvent)`
  - `Latest(n int) []NewsItem`

### 4.3 `/internal/news/service`
- `NewsService` owns:
  - generator loop (optional): can be stubbed as manual `Publish(item)` for now.
  - authoritative internal channel feeding view
  - external channel for UI/traders (droppable configurable)
- Methods:
  - `Publish(item NewsItem)` (sets ID/time if missing)
  - `Latest(n int) []NewsItem` (from view)
  - `Events() <-chan NewsEvent`
  - `Close()`

Config:
- `TapeSize`, buffers, drop flag.

---

## 5) Trader subsystem (strategy + runner)

### 5.1 `/internal/trader/types.go`
Define:
- `TraderID int64`
- `OrderIntent`:
  - `TickerID`, `Kind`, `Side`, `Price`, `Size`
- `TraderEvent`:
  - `TraderID`, `Time`, `Type` (PlacedOrder/RequestedApproval/etc.), payload.

### 5.2 `/internal/trader/strategy/interface.go`
Define interfaces (owned by trader package):
- `type MarketReader interface { Snapshot() marketview.MarketSnapshot; GetLevels(tid, side) ...; GetTradesLast(tid, n) ... }`
- `type NewsReader interface { Latest(n int) []news.NewsItem }`
- `type OrderSender interface { SubmitLimit(...); SubmitMarket(...); Cancel(...) }`

Define:
- `type Strategy interface { Step(ctx, now, mr, nr) ([]OrderIntent, []TraderEvent) }`

Provide `example_strategy.go` implementing trivial behavior (optional).

### 5.3 `/internal/trader/runner`
Implement runner that:
- Accepts:
  - `Strategy`
  - `MarketReader` + `OrderSender` (can be same market service)
  - `NewsReader`
- Loop:
  - ticks on `time.Ticker` (configurable)
  - calls `Strategy.Step()`
  - sends orders via `OrderSender`
  - publishes `TraderEvent`s to external channel (droppable configurable)
- Methods:
  - `Events() <-chan TraderEvent`
  - `Close()`

---

## 6) Broker subsystem (player orchestration; minimal stub)

- Broker subscribes to trader events and news; exposes view of “requests”.
- Initial implementation can be minimal:
  - `BrokerService` with:
    - `AttachTraderEvents(<-chan TraderEvent)`
    - `AttachNewsEvents(<-chan news.NewsEvent)`
    - `Requests()` snapshot getter from broker view
- No trading logic required now.

---

## 7) Game wiring `/internal/game`

Implement `Game` struct that owns:
- `MarketService`
- `NewsService`
- `[]TraderRunner`
- `BrokerService` (optional)
- `Close()` shuts down in reverse dependency order:
  - stop traders
  - stop news
  - stop market
  - stop broker

Provide `NewGame(cfg)` that:
- creates market with tickers
- creates news
- creates traders and passes market/news interfaces
- attaches trader/news events to broker if enabled

---

## 8) TUI entrypoint `/cmd/tui/main.go` (wiring only)
- Construct `game.NewGame()`
- Subscribe to:
  - `market.Events()`, `news.Events()`, `trader.Events()`, (broker events if any)
- Redraw policy:
  - UI event loop triggers refresh on any event (debounce optional)
  - UI renders via snapshots:
    - `market.Snapshot()`
    - selected book: `GetLevels`, `GetOrders`, `GetTradesLast`
    - `news.Latest(n)`
    - trader events buffer (runner/broker view)

---

## Implementation plan (ordered steps)

1) **Refactor orderbook packages**
- Move existing code into `/internal/orderbook/core`, `/internal/orderbook/view`, `/internal/orderbook/service`.
- Update package names/imports accordingly.
- Ensure `go test ./...` compiles.

2) **Enforce no pointer leakage**
- Verify no public API returns pointers to internal orders/trades.
- Ensure view returns values and deep-copies slices/maps.

3) **Add Market**
- Implement `/internal/market/types.go`, `/internal/market/view`, `/internal/market/service`.
- Write a small compile-time example in a temporary test or `internal/market/service_test.go`:
  - create 2 tickers
  - submit orders into one ticker
  - confirm `Snapshot().ByTicker[tid].BidOK/AskOK` updates and `GetTradesLast` works.

4) **Add News (minimal)**
- Implement news types/view/service with manual `Publish()`.
- Add simple test verifying `Latest(n)` ordering and capacity.

5) **Add Trader interfaces + runner skeleton**
- Implement strategy interfaces and a trivial strategy (e.g., place a small bid if spread exists).
- Implement runner tick loop + `Events()` channel.
- Add test with mocked `MarketReader/OrderSender` to ensure runner calls sender.

6) **Add Game wiring**
- Implement `internal/game` to wire market/news/traders.
- Ensure `Close()` reliably terminates goroutines (use WaitGroups).

7) **TUI stub**
- Implement `/cmd/tui/main.go` that starts game and prints snapshots periodically or on events (no UI library required yet).
- Ensure graceful exit on SIGINT.

8) **Concurrency and shutdown verification**
- Run with `-race`.
- Ensure no goroutine leaks (all loops select on `closed`).

---

## Hard constraints for the agent

- No circular imports.
- All goroutine loops must terminate on `Close()`; use `WaitGroup`.
- External event channels may be droppable, but internal view-updating channels must not drop.
- Maps must be keyed by `TickerID`, not `Ticker`.
- Views must return copies (no internal references).
- `orderbook/core` must not call `time.Now()` or generate IDs.

---

## Acceptance checks

- `go test ./...` passes.
- `go test -race ./...` passes (no data races).
- Market can:
  - submit orders to different tickers independently
  - provide per-ticker book levels and last trades
  - provide aggregated snapshot across tickers
- News can publish and retrieve latest items.
- Trader runner can place orders via market service using snapshots.
- Game can start/close without deadlocks.