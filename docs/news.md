# News System

The news system provides a publish/subscribe mechanism for delivering news items to all subscribers with configurable delivery windows.

## Package Structure

```
/internal/news
  types.go              # NewsItem, Severity
  /view
    events.go           # NewsPublished event
    view.go             # Ring buffer view of news
  /service
    config.go           # Service configuration
    service.go          # News publisher service
```

## Types

```go
type Severity uint8
const (
    SeverityInfo Severity = iota
    SeverityWarning
    SeverityCritical
)

type NewsItem struct {
    ID        int64
    Headline  string
    Body      string
    Severity  Severity
    Timestamp int64  // Unix nanos
}
```

## View Package (`/internal/news/view`)

### Events

```go
type NewsEvent interface {
    isNewsEvent()
}

type NewsPublished struct {
    Item NewsItem
}
```

### NewsView

Ring buffer that stores recent news:

```go
type NewsView struct { ... }

func NewNewsView(capacity int) *NewsView
func (v *NewsView) Apply(ev NewsEvent)
func (v *NewsView) Recent(n int) []NewsItem  // Returns copies, newest first
func (v *NewsView) All() []NewsItem           // Returns all, newest first
```

**Thread Safety:**
- Uses `sync.RWMutex`
- `Apply()` takes write lock
- `Recent()` / `All()` take read lock
- Returns are deep copies

## Service Package (`/internal/news/service`)

### Configuration

```go
type Config struct {
    Capacity    int   // Ring buffer size (default: 100)
    EventBuffer int   // Event channel size (default: 256)
    DropEvents  bool  // Drop events on overflow (default: true)
}
```

### NewsService

```go
type NewsService struct { ... }

func NewNewsService(cfg Config) *NewsService

// Publish news (generates ID and timestamp)
func (s *NewsService) Publish(headline, body string, severity Severity) NewsItem

// View access
func (s *NewsService) Recent(n int) []NewsItem
func (s *NewsService) All() []NewsItem

// Events
func (s *NewsService) Events() <-chan view.NewsEvent

// Lifecycle
func (s *NewsService) Close()
```

### Internal Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                        NewsService                             │
│                                                                │
│  Publish()                                                     │
│     │                                                          │
│     ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  Generate ID (atomic.Int64)                              │  │
│  │  Set timestamp (time.Now().UnixNano())                   │  │
│  │  Create NewsItem                                         │  │
│  └───────────────────────┬─────────────────────────────────┘  │
│                          │                                     │
│                          ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  cmdCh <- publishCmd{item}                               │  │
│  └───────────────────────┬─────────────────────────────────┘  │
│                          │                                     │
│                          ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │               Process Goroutine                          │  │
│  │  for cmd := range cmdCh {                                │  │
│  │    ev := NewsPublished{cmd.item}                        │  │
│  │    view.Apply(ev)                                        │  │
│  │    select {                                              │  │
│  │      case events <- ev:                                  │  │
│  │      default: dropped++                                  │  │
│  │    }                                                     │  │
│  │  }                                                       │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                │
│  State:                                                        │
│    view   *view.NewsView   // Ring buffer                     │
│    idGen  atomic.Int64     // ID generator                    │
│                                                                │
└───────────────────────────────────────────────────────────────┘
```

### ID Generation

Like the orderbook service:
- Uses `atomic.Int64` initialized from `time.Now().UnixNano()`
- Each `Publish()` gets `idGen.Add(1)`
- Avoids collisions across restarts

## Ring Buffer Implementation

The `NewsView` uses a ring buffer for O(1) append and bounded memory:

```
capacity = 5

Initial state:
┌───┬───┬───┬───┬───┐
│   │   │   │   │   │  head=0, count=0
└───┴───┴───┴───┴───┘

After 3 publishes:
┌───┬───┬───┬───┬───┐
│ A │ B │ C │   │   │  head=0, count=3
└───┴───┴───┴───┴───┘

After 6 publishes (wraps):
┌───┬───┬───┬───┬───┐
│ F │ B │ C │ D │ E │  head=1, count=5
└───┴───┴───┴───┴───┘
  ▲                    (A was overwritten by F)
  └── next write position

Recent(3) returns: [F, E, D] (newest first)
```

## Usage Example

```go
// Create news service
cfg := service.Config{
    Capacity:    100,
    EventBuffer: 256,
}
news := service.NewNewsService(cfg)
defer news.Close()

// Subscribe to news events
go func() {
    for ev := range news.Events() {
        switch e := ev.(type) {
        case view.NewsPublished:
            fmt.Printf("[%s] %s\n", 
                severityStr(e.Item.Severity),
                e.Item.Headline)
        }
    }
}()

// Publish news
news.Publish(
    "AAPL Earnings Beat Expectations",
    "Apple Inc. reported quarterly earnings...",
    SeverityInfo,
)

news.Publish(
    "Market Volatility Alert",
    "Unusual trading activity detected...",
    SeverityWarning,
)

// Query recent news
for _, item := range news.Recent(5) {
    fmt.Printf("%d: %s\n", item.ID, item.Headline)
}
```

## Integration with Game

The `Game` struct owns the `NewsService` and uses it for:

1. **Market Events**: Publishing significant price moves
2. **Trading Alerts**: Large orders, unusual activity
3. **Game Events**: Round start/end, time warnings
4. **System Messages**: Connection status, errors

```go
// In game loop
if priceMove > threshold {
    game.news.Publish(
        fmt.Sprintf("%s moves %.1f%%", ticker, priceMove*100),
        "Large price movement detected",
        news.SeverityWarning,
    )
}
```

## Design Decisions

### Why Ring Buffer?

- **Bounded Memory**: Fixed maximum news count
- **Fast Append**: O(1) insertion
- **Natural Expiry**: Oldest news automatically removed
- **Simple Implementation**: Just index math

### Why Synchronous View Updates?

Unlike external event dispatch (which may drop):
- `view.Apply()` is always called
- Ensures local state is consistent
- Only external subscribers may miss events

### Severity Levels

Three levels cover most cases:
- **Info**: Normal market updates, routine events
- **Warning**: Unusual activity, price alerts
- **Critical**: System errors, emergency halts

UI can filter or highlight based on severity.
