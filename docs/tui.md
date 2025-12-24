# TUI (Terminal User Interface)

The TUI application provides a real-time terminal interface for viewing market data, order books, news, and broker activity.

## Location

```
/cmd/tui/main.go
```

## Dependencies

The TUI uses the [Charm](https://charm.sh/) ecosystem:
- **Bubble Tea**: Elm-architecture TUI framework
- **Lip Gloss**: Styling and layout

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              TUI Application                             │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                         Bubble Tea Program                          │ │
│  │                                                                      │ │
│  │   tea.NewProgram(model)                                             │ │
│  │     │                                                                │ │
│  │     ├── Init() → start tick cmd                                     │ │
│  │     │                                                                │ │
│  │     ├── Update(msg) → handle keys, ticks                           │ │
│  │     │                                                                │ │
│  │     └── View() → render UI                                          │ │
│  │                                                                      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                            Model                                    │ │
│  │                                                                      │ │
│  │   game     *game.Game      // Game instance                         │ │
│  │   selected TickerID        // Currently selected ticker             │ │
│  │   width    int             // Terminal width                        │ │
│  │   height   int             // Terminal height                       │ │
│  │                                                                      │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## UI Layout

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Stockcraft                                  │
│                          Time Remaining: 4:32                           │
├───────────────────────────────────┬─────────────────────────────────────┤
│         Market Overview           │            Order Book               │
│  ┌─────────────────────────────┐  │  ┌───────────────────────────────┐  │
│  │ AAPL   150.25  +0.50 (+0.3%)│  │  │ ASKS                          │  │
│  │ GOOG  2845.00  -12.00(-0.4%)│  │  │   151.00    500               │  │
│  │ MSFT   415.75  +2.25 (+0.5%)│  │  │   150.75    250               │  │
│  └─────────────────────────────┘  │  │   150.50    100               │  │
│                                   │  │ ─────────────────────────────  │  │
│                                   │  │   150.25    150    ← spread    │  │
│                                   │  │ ─────────────────────────────  │  │
│                                   │  │   150.00    200               │  │
│                                   │  │   149.75    300               │  │
│                                   │  │   149.50    450               │  │
│                                   │  │ BIDS                          │  │
│                                   │  └───────────────────────────────┘  │
├───────────────────────────────────┼─────────────────────────────────────┤
│             News                  │         Broker Requests             │
│  ┌─────────────────────────────┐  │  ┌───────────────────────────────┐  │
│  │ [INFO] AAPL +0.5%           │  │  │ #1234 BUY AAPL 100@150.00    │  │
│  │ [WARN] High volatility      │  │  │ #1235 SELL GOOG 50@2845.00   │  │
│  │ [INFO] Game started         │  │  │ #1236 BUY MSFT 200@415.50    │  │
│  └─────────────────────────────┘  │  └───────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────────────┤
│  [Q]uit  [↑↓]Select Ticker  [Tab]Next Section                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Components

### Market Panel

Displays all tickers with current prices and changes:

```go
func (m model) renderMarket() string {
    snapshots := m.game.Market().AllSnapshots()
    
    var rows []string
    for _, ticker := range m.game.Market().Tickers() {
        snap := snapshots[ticker]
        
        // Calculate change from previous
        change := snap.LastPrice - m.prevPrices[ticker]
        pct := float64(change) / float64(m.prevPrices[ticker]) * 100
        
        style := priceStyle
        if change > 0 {
            style = style.Foreground(lipgloss.Color("green"))
        } else if change < 0 {
            style = style.Foreground(lipgloss.Color("red"))
        }
        
        rows = append(rows, fmt.Sprintf("%s  %s  %+.2f (%+.1f%%)",
            ticker, formatPrice(snap.LastPrice), float64(change)/100, pct))
    }
    
    return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
```

### Order Book Panel

Shows bid/ask levels for selected ticker:

```go
func (m model) renderOrderBook() string {
    asks := m.game.Market().GetLevels(m.selected, core.SideSell)
    bids := m.game.Market().GetLevels(m.selected, core.SideBuy)
    
    var lines []string
    
    // Asks (top to bottom, highest first)
    lines = append(lines, askHeader)
    for i := len(asks) - 1; i >= 0; i-- {
        lines = append(lines, formatLevel(asks[i], "red"))
    }
    
    // Spread line
    spread := asks[0].Price - bids[0].Price
    lines = append(lines, fmt.Sprintf("── spread: %d ──", spread))
    
    // Bids (top to bottom, highest first)
    for _, lvl := range bids {
        lines = append(lines, formatLevel(lvl, "green"))
    }
    lines = append(lines, bidHeader)
    
    return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
```

### News Panel

Shows recent news with severity coloring:

```go
func (m model) renderNews() string {
    items := m.game.News().Recent(5)
    
    var lines []string
    for _, item := range items {
        prefix := "[INFO]"
        style := infoStyle
        
        switch item.Severity {
        case news.SeverityWarning:
            prefix = "[WARN]"
            style = warnStyle
        case news.SeverityCritical:
            prefix = "[CRIT]"
            style = critStyle
        }
        
        lines = append(lines, style.Render(fmt.Sprintf("%s %s", prefix, item.Headline)))
    }
    
    return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
```

### Broker Panel

Shows pending and recent broker requests:

```go
func (m model) renderBroker() string {
    pending := m.game.Broker().Pending()
    
    var lines []string
    for _, req := range pending {
        typeStr := "BUY"
        if req.Type == broker.RequestTypeSell {
            typeStr = "SELL"
        }
        
        lines = append(lines, fmt.Sprintf("#%d %s %s %d@%d",
            req.ID, typeStr, req.Ticker, req.Size, req.Price))
    }
    
    return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
```

## Key Bindings

| Key | Action |
|-----|--------|
| `q`, `Ctrl+C` | Quit |
| `↑` / `k` | Select previous ticker |
| `↓` / `j` | Select next ticker |
| `Tab` | Focus next panel |
| `Shift+Tab` | Focus previous panel |

## Update Loop

The TUI updates on a tick interval:

```go
type tickMsg time.Time

func tickCmd() tea.Cmd {
    return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "up", "k":
            m.selectPrevTicker()
        case "down", "j":
            m.selectNextTicker()
        }
        
    case tickMsg:
        // Refresh data from game
        return m, tickCmd()
        
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    
    return m, nil
}
```

## Running the TUI

```bash
# Run directly
go run ./cmd/tui

# Build and run
go build -o stockcraft ./cmd/tui
./stockcraft
```

## Styling

Uses Lip Gloss for consistent styling:

```go
var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("white")).
        Background(lipgloss.Color("blue")).
        Padding(0, 1)
    
    panelStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("gray")).
        Padding(1)
    
    selectedStyle = lipgloss.NewStyle().
        Bold(true).
        Background(lipgloss.Color("blue"))
    
    priceUpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("green"))
    
    priceDownStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("red"))
)
```

## Thread Safety

The TUI reads from game subsystems which are all thread-safe:
- Market snapshots: `sync.RWMutex` protected
- News items: `sync.RWMutex` protected
- Broker requests: `sync.RWMutex` protected

All reads return copies, so the TUI can safely render without locks.
