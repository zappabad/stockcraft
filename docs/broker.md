# Broker System

The broker system provides a placeholder for player orchestration, handling the interaction between the game and human players or external systems.

## Package Structure

```
/internal/broker
  types.go              # BrokerRequest types
  /view
    view.go             # Pending request queue view
  /service
    config.go           # Service configuration
    service.go          # Broker service implementation
```

## Types

```go
type RequestType uint8
const (
    RequestTypeBuy RequestType = iota
    RequestTypeSell
    RequestTypeCancel
)

type BrokerRequest struct {
    ID        int64
    UserID    int64
    Type      RequestType
    Ticker    TickerID
    Price     int64   // For limit orders
    Size      int64
    Timestamp int64
    Status    RequestStatus
}

type RequestStatus uint8
const (
    StatusPending RequestStatus = iota
    StatusAccepted
    StatusRejected
    StatusCancelled
)
```

## View Package (`/internal/broker/view`)

### BrokerView

Maintains a view of pending and recent requests:

```go
type BrokerView struct { ... }

func NewBrokerView(capacity int) *BrokerView
func (v *BrokerView) Apply(ev BrokerEvent)

// Query methods
func (v *BrokerView) Pending() []BrokerRequest
func (v *BrokerView) Recent(n int) []BrokerRequest
func (v *BrokerView) ByUser(userID int64) []BrokerRequest
```

**Thread Safety:**
- Uses `sync.RWMutex`
- Query methods return copies

## Service Package (`/internal/broker/service`)

### Configuration

```go
type Config struct {
    RequestQueueSize int   // Max pending requests (default: 1000)
    HistorySize      int   // Recent request history (default: 100)
    EventBuffer      int   // Event channel size (default: 256)
}
```

### BrokerService

```go
type BrokerService struct { ... }

func NewBrokerService(cfg Config) *BrokerService

// Submit requests (from players/external systems)
func (s *BrokerService) SubmitRequest(req BrokerRequest) error

// Request processing (called by game loop)
func (s *BrokerService) NextRequest() (BrokerRequest, bool)
func (s *BrokerService) CompleteRequest(id int64, status RequestStatus)

// View access
func (s *BrokerService) Pending() []BrokerRequest
func (s *BrokerService) Recent(n int) []BrokerRequest

// Events
func (s *BrokerService) Events() <-chan BrokerEvent

// Lifecycle
func (s *BrokerService) Close()
```

### Current Implementation Status

**Note**: The broker system is currently a minimal stub. The implementation provides:

1. **Basic request queue**: Stores pending requests
2. **Event publication**: Publishes request events
3. **View synchronization**: Maintains read-only view

**Not yet implemented**:
- Request validation
- Rate limiting
- Authentication/authorization
- Multi-player coordination
- Leaderboard integration

## Architecture

```
┌───────────────────────────────────────────────────────────────────┐
│                        BrokerService                               │
│                                                                    │
│  External API                        Internal Processing           │
│  ┌─────────────────┐                 ┌─────────────────┐          │
│  │ SubmitRequest() │────────────────►│  requestQueue   │          │
│  └─────────────────┘                 │  (chan or slice)│          │
│                                      └────────┬────────┘          │
│                                               │                    │
│  Game Loop                                    │                    │
│  ┌─────────────────┐                         │                    │
│  │ NextRequest()   │◄────────────────────────┘                    │
│  └────────┬────────┘                                              │
│           │                                                        │
│           ▼                                                        │
│  ┌─────────────────┐     ┌─────────────────┐                      │
│  │ Process via     │────►│ CompleteRequest │                      │
│  │ MarketService   │     │ (update status) │                      │
│  └─────────────────┘     └────────┬────────┘                      │
│                                   │                                │
│                                   ▼                                │
│                          ┌─────────────────┐                      │
│                          │   BrokerView    │                      │
│                          │ (pending/recent)│                      │
│                          └─────────────────┘                      │
│                                                                    │
└───────────────────────────────────────────────────────────────────┘
```

## Integration with Game

The broker serves as the interface between external players and the trading system:

```go
// In game loop
func (g *Game) processRequests() {
    for {
        req, ok := g.broker.NextRequest()
        if !ok {
            break  // No more pending requests
        }
        
        var err error
        switch req.Type {
        case broker.RequestTypeBuy:
            _, err = g.market.SubmitLimit(ctx, req.Ticker, req.UserID,
                core.SideBuy, req.Price, req.Size)
        case broker.RequestTypeSell:
            _, err = g.market.SubmitLimit(ctx, req.Ticker, req.UserID,
                core.SideSell, req.Price, req.Size)
        case broker.RequestTypeCancel:
            _, err = g.market.Cancel(ctx, req.Ticker, req.OrderID)
        }
        
        status := broker.StatusAccepted
        if err != nil {
            status = broker.StatusRejected
        }
        g.broker.CompleteRequest(req.ID, status)
    }
}
```

## Future Extensions

The broker system is designed to be extended for:

### 1. HTTP/WebSocket API

```go
// Future: REST endpoint
func (s *BrokerService) HandleOrder(w http.ResponseWriter, r *http.Request) {
    var req BrokerRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    if err := s.ValidateRequest(req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    s.SubmitRequest(req)
    json.NewEncoder(w).Encode(map[string]int64{"id": req.ID})
}
```

### 2. Authentication

```go
type AuthenticatedRequest struct {
    BrokerRequest
    Token     string
    Signature string
}

func (s *BrokerService) Authenticate(req AuthenticatedRequest) error {
    // Verify token and signature
}
```

### 3. Rate Limiting

```go
type RateLimiter struct {
    limits map[int64]*tokenBucket  // Per-user limits
}

func (s *BrokerService) SubmitRequest(req BrokerRequest) error {
    if !s.rateLimiter.Allow(req.UserID) {
        return ErrRateLimited
    }
    // ...
}
```

### 4. Multi-Player Coordination

```go
type Session struct {
    ID        string
    Players   []int64
    StartTime int64
    Duration  time.Duration
    State     SessionState
}

func (s *BrokerService) JoinSession(sessionID string, userID int64) error
func (s *BrokerService) LeaveSession(sessionID string, userID int64) error
```

## Usage Example

```go
// Create broker service
cfg := service.Config{
    RequestQueueSize: 1000,
    HistorySize:      100,
}
broker := service.NewBrokerService(cfg)
defer broker.Close()

// Submit a request (from player/API)
req := BrokerRequest{
    UserID:    playerID,
    Type:      RequestTypeBuy,
    Ticker:    "AAPL",
    Price:     150_00,  // $150.00
    Size:      100,
}
broker.SubmitRequest(req)

// Process in game loop
for {
    req, ok := broker.NextRequest()
    if !ok {
        break
    }
    
    // Execute via market service
    _, err := market.SubmitLimit(ctx, req.Ticker, req.UserID,
        core.SideBuy, req.Price, req.Size)
    
    status := StatusAccepted
    if err != nil {
        status = StatusRejected
    }
    broker.CompleteRequest(req.ID, status)
}

// Query state
pending := broker.Pending()
fmt.Printf("Pending requests: %d\n", len(pending))
```

## Design Decisions

### Why a Separate Broker?

- **Separation of Concerns**: Market handles matching, broker handles player interaction
- **Extensibility**: Easy to add auth, rate limiting, sessions
- **Testability**: Can test player logic independently of market logic

### Why Request Queue?

- **Fairness**: FIFO processing of player requests
- **Back-pressure**: Queue size limits prevent overload
- **Async Processing**: Players don't block on order execution

### Minimal Initial Implementation

The broker is intentionally minimal because:
- Core trading logic is in market/orderbook
- Player interaction needs may vary (HTTP, WebSocket, direct)
- Better to extend than remove features
