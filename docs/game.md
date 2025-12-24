# Game System

The game system is the top-level composition layer that owns and coordinates all subsystems (market, news, trader, broker) and manages the game lifecycle.

## Package Structure

```
/internal/game
  config.go     # Game configuration
  game.go       # Game struct and lifecycle
```

## Configuration

```go
type Config struct {
    // Tickers to trade
    Tickers []TickerID
    
    // Subsystem configs
    MarketConfig  marketservice.Config
    NewsConfig    newsservice.Config
    BrokerConfig  brokerservice.Config
    
    // Trader configs (one per bot)
    Traders []TraderConfig
    
    // Game rules
    Duration      time.Duration  // Game length (default: 5 minutes)
    TickInterval  time.Duration  // Main loop tick (default: 100ms)
}

type TraderConfig struct {
    Strategy     strategy.Strategy
    RunnerConfig runner.Config
}
```

## Game API

```go
type Game struct { ... }

func NewGame(cfg Config) *Game

// Lifecycle
func (g *Game) Start() error
func (g *Game) Stop()
func (g *Game) Wait()  // Block until game ends

// State
func (g *Game) IsRunning() bool
func (g *Game) TimeRemaining() time.Duration
func (g *Game) Results() GameResults

// Subsystem access (read-only views)
func (g *Game) Market() *marketservice.MarketService
func (g *Game) News() *newsservice.NewsService
func (g *Game) Broker() *brokerservice.BrokerService
```

## Internal Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                               Game                                       │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                         Owned Subsystems                            │ │
│  │                                                                      │ │
│  │   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐         │ │
│  │   │MarketService │    │ NewsService  │    │BrokerService │         │ │
│  │   │              │    │              │    │              │         │ │
│  │   │ - AAPL book  │    │ - Ring buf   │    │ - Request Q  │         │ │
│  │   │ - GOOG book  │    │ - Events     │    │ - History    │         │ │
│  │   │ - MSFT book  │    │              │    │              │         │ │
│  │   └──────────────┘    └──────────────┘    └──────────────┘         │ │
│  │                                                                      │ │
│  │   ┌──────────────────────────────────────────────────────────────┐ │ │
│  │   │                    TraderRunners[]                            │ │ │
│  │   │   ┌────────────┐  ┌────────────┐  ┌────────────┐             │ │ │
│  │   │   │ Bot 1      │  │ Bot 2      │  │ Bot 3      │             │ │ │
│  │   │   │ Strategy A │  │ Strategy B │  │ Strategy C │             │ │ │
│  │   │   └────────────┘  └────────────┘  └────────────┘             │ │ │
│  │   └──────────────────────────────────────────────────────────────┘ │ │
│  │                                                                      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                          Game Loop                                  │ │
│  │   ticker := time.NewTicker(cfg.TickInterval)                       │ │
│  │   for {                                                             │ │
│  │     select {                                                        │ │
│  │     case <-stopCh:                                                 │ │
│  │       shutdown()                                                    │ │
│  │       return                                                        │ │
│  │     case <-gameTimer.C:                                            │ │
│  │       publishNews("Game Over")                                      │ │
│  │       shutdown()                                                    │ │
│  │       return                                                        │ │
│  │     case <-ticker.C:                                               │ │
│  │       processBrokerRequests()                                       │ │
│  │       checkGameEvents()                                             │ │
│  │     }                                                               │ │
│  │   }                                                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Lifecycle

### Startup Sequence

```
NewGame(cfg)
    │
    ├── Create MarketService(cfg.MarketConfig)
    │     └── For each ticker: create orderbook.Service
    │
    ├── Create NewsService(cfg.NewsConfig)
    │
    ├── Create BrokerService(cfg.BrokerConfig)
    │
    └── For each TraderConfig:
          └── Create TraderRunner(strategy, market)

Start()
    │
    ├── Start all TraderRunners
    │     └── Each starts tick + fill listener goroutines
    │
    ├── Start game loop goroutine
    │     ├── Process broker requests
    │     └── Publish time warnings
    │
    └── Publish "Game Started" news
```

### Shutdown Sequence

```
Stop() or game timer expires
    │
    ├── Signal stop (close stopCh)
    │
    ├── Stop all TraderRunners
    │     └── Each stops tick goroutine, waits for fills
    │
    ├── Wait for game loop to exit
    │
    ├── Close BrokerService
    │
    ├── Close NewsService
    │
    ├── Close MarketService
    │     └── Each orderbook.Service closes
    │
    ├── Compute final results
    │
    └── Mark game as stopped
```

### Graceful Shutdown

Uses `sync.WaitGroup` to ensure clean shutdown:

```go
func (g *Game) shutdown() {
    // 1. Stop traders first (they submit orders)
    for _, t := range g.traders {
        t.Stop()
    }
    
    // 2. Process remaining broker requests
    g.drainBrokerQueue()
    
    // 3. Wait for in-flight orders to complete
    g.wg.Wait()
    
    // 4. Close services (stops goroutines)
    g.broker.Close()
    g.news.Close()
    g.market.Close()
    
    // 5. Compute results
    g.results = g.computeResults()
}
```

## Game Loop

The main game loop runs on a tick interval:

```go
func (g *Game) loop() {
    ticker := time.NewTicker(g.cfg.TickInterval)
    defer ticker.Stop()
    
    gameEnd := time.NewTimer(g.cfg.Duration)
    defer gameEnd.Stop()
    
    // Time warnings at 1 min and 10 sec
    warnings := []time.Duration{
        g.cfg.Duration - time.Minute,
        g.cfg.Duration - 10*time.Second,
    }
    
    for {
        select {
        case <-g.stopCh:
            return
            
        case <-gameEnd.C:
            g.news.Publish("Game Over", "Final results being calculated", SeverityCritical)
            return
            
        case <-ticker.C:
            // Process broker requests
            g.processBrokerRequests()
            
            // Check for time warnings
            remaining := g.TimeRemaining()
            for _, warn := range warnings {
                if remaining <= warn && !g.warningSent[warn] {
                    g.news.Publish(
                        fmt.Sprintf("%s remaining", remaining),
                        "Time warning",
                        SeverityWarning,
                    )
                    g.warningSent[warn] = true
                }
            }
        }
    }
}
```

## Results

```go
type GameResults struct {
    Duration    time.Duration
    FinalPrices map[TickerID]int64
    
    TraderResults []TraderResult
}

type TraderResult struct {
    Name      string
    UserID    int64
    PnL       int64
    Trades    int
    Portfolio map[TickerID]int64
}
```

Results are computed at shutdown:

```go
func (g *Game) computeResults() GameResults {
    results := GameResults{
        Duration:    g.cfg.Duration,
        FinalPrices: make(map[TickerID]int64),
    }
    
    // Get final prices
    for ticker, snap := range g.market.AllSnapshots() {
        results.FinalPrices[ticker] = snap.LastPrice
    }
    
    // Get trader results
    for _, t := range g.traders {
        results.TraderResults = append(results.TraderResults, TraderResult{
            Name:      t.Strategy().Name(),
            UserID:    t.UserID(),
            PnL:       t.PnL(),
            Portfolio: t.Portfolio(),
        })
    }
    
    return results
}
```

## Usage Example

```go
// Configure game
cfg := game.Config{
    Tickers: []TickerID{"AAPL", "GOOG", "MSFT"},
    MarketConfig: marketservice.Config{
        OrderBookConfig: observice.DefaultConfig(),
        EventBuffer:     1024,
    },
    NewsConfig: newsservice.Config{
        Capacity: 100,
    },
    BrokerConfig: brokerservice.Config{
        RequestQueueSize: 1000,
    },
    Traders: []game.TraderConfig{
        {
            Strategy: strategy.NewExampleStrategy(
                []TickerID{"AAPL", "GOOG", "MSFT"},
                10, 100,
            ),
            RunnerConfig: runner.Config{
                TickInterval: 100 * time.Millisecond,
                UserID:       1,
                InitialCash:  1_000_000,
            },
        },
        // Add more traders...
    },
    Duration:     5 * time.Minute,
    TickInterval: 100 * time.Millisecond,
}

// Create and start game
g := game.NewGame(cfg)
if err := g.Start(); err != nil {
    log.Fatal(err)
}

// Wait for game to end (or call g.Stop() to end early)
g.Wait()

// Get results
results := g.Results()
fmt.Printf("Game lasted %s\n", results.Duration)
for _, tr := range results.TraderResults {
    fmt.Printf("%s: P&L = %d\n", tr.Name, tr.PnL)
}
```

## TUI Integration

The `cmd/tui/main.go` creates a Game and displays its state:

```go
func main() {
    cfg := game.Config{
        Tickers:      []TickerID{"AAPL", "GOOG", "MSFT"},
        MarketConfig: marketservice.DefaultConfig(),
        // ...
    }
    
    g := game.NewGame(cfg)
    g.Start()
    defer g.Stop()
    
    // Create TUI with game references
    p := tea.NewProgram(newModel(g))
    p.Run()
}

type model struct {
    game *game.Game
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Query game state for display
    snap := m.game.Market().Snapshot("AAPL")
    news := m.game.News().Recent(5)
    // ...
}
```

## Design Decisions

### Single Owner

The Game struct is the single owner of all subsystems:
- Creates them in `NewGame`
- Starts them in `Start`
- Closes them in `Stop`
- No shared ownership or reference counting

### Composition Over Inheritance

The Game doesn't extend any subsystem; it composes them:
- Clear ownership boundaries
- Easy to test each subsystem independently
- No diamond inheritance problems

### Tick-Based Main Loop

The game loop uses a ticker rather than event-driven updates:
- Predictable timing for game events
- Easier to reason about time warnings
- Broker requests processed in batches

### Results at Shutdown

Results are computed once at shutdown:
- Consistent snapshot of final state
- No need for real-time P&L (that's per-trader)
- Avoids race conditions with ongoing trading
