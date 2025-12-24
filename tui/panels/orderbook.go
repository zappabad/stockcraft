package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zappabad/stockcraft/internal/market"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
	orderbookview "github.com/zappabad/stockcraft/internal/orderbook/view"
	"github.com/zappabad/stockcraft/tui/styles"
)

// OrderbookPanel displays the orderbook for a selected ticker.
type OrderbookPanel struct {
	ticker       market.Ticker
	bids         []orderbookview.Level
	asks         []orderbookview.Level
	trades       []core.TradeEvent
	scrollOffset int
	focused      bool
	width        int
	height       int
	maxLevels    int
}

// NewOrderbookPanel creates a new orderbook panel.
func NewOrderbookPanel() *OrderbookPanel {
	return &OrderbookPanel{
		maxLevels: 10,
	}
}

// Init initializes the panel.
func (p *OrderbookPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the panel.
func (p *OrderbookPanel) Update(msg tea.Msg) (*OrderbookPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if p.scrollOffset > 0 {
				p.scrollOffset--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			p.scrollOffset++
		}
	}
	return p, nil
}

// View renders the panel.
func (p *OrderbookPanel) View() string {
	var content strings.Builder

	// Title with ticker name
	tickerName := "No ticker selected"
	if p.ticker.Name != "" {
		tickerName = p.ticker.Name
	}

	// Calculate available height for orders
	availableHeight := p.height - 6 // Account for header, title, borders
	levelsToShow := availableHeight / 2
	if levelsToShow > p.maxLevels {
		levelsToShow = p.maxLevels
	}
	if levelsToShow < 3 {
		levelsToShow = 3
	}

	// Header
	header := fmt.Sprintf("%10s %8s │ %8s %10s", "BidSz", "Bid", "Ask", "AskSz")
	content.WriteString(styles.HeaderStyle.Render(header))
	content.WriteString("\n")

	// Get levels to display
	bidsToShow := p.bids
	if len(bidsToShow) > levelsToShow {
		bidsToShow = bidsToShow[:levelsToShow]
	}
	asksToShow := p.asks
	if len(asksToShow) > levelsToShow {
		asksToShow = asksToShow[:levelsToShow]
	}

	// Find max rows needed
	maxRows := len(bidsToShow)
	if len(asksToShow) > maxRows {
		maxRows = len(asksToShow)
	}

	// Render side by side
	for i := 0; i < maxRows; i++ {
		bidSize := ""
		bidPrice := ""
		askPrice := ""
		askSize := ""

		if i < len(bidsToShow) {
			bidSize = fmt.Sprintf("%d", bidsToShow[i].Size)
			bidPrice = formatPrice(int64(bidsToShow[i].Price), p.ticker.Decimals)
		}
		if i < len(asksToShow) {
			askPrice = formatPrice(int64(asksToShow[i].Price), p.ticker.Decimals)
			askSize = fmt.Sprintf("%d", asksToShow[i].Size)
		}

		bidPart := fmt.Sprintf("%10s %8s", bidSize, bidPrice)
		askPart := fmt.Sprintf("%8s %10s", askPrice, askSize)

		bidStyled := styles.BuyStyle.Render(bidPart)
		askStyled := styles.SellStyle.Render(askPart)

		content.WriteString(fmt.Sprintf("%s │ %s\n", bidStyled, askStyled))
	}

	// Recent trades section
	content.WriteString("\n")
	content.WriteString(styles.HeaderStyle.Render("Recent Trades"))
	content.WriteString("\n")

	tradesToShow := p.trades
	if len(tradesToShow) > 5 {
		tradesToShow = tradesToShow[len(tradesToShow)-5:]
	}

	for _, trade := range tradesToShow {
		price := formatPrice(int64(trade.Price), p.ticker.Decimals)
		size := fmt.Sprintf("%d", trade.Size)

		var sideStyle lipgloss.Style
		if trade.TakerSide == core.SideBuy {
			sideStyle = styles.BuyStyle
		} else {
			sideStyle = styles.SellStyle
		}

		tradeStr := fmt.Sprintf("%8s @ %8s", size, price)
		content.WriteString(sideStyle.Render(tradeStr))
		content.WriteString("\n")
	}

	// Apply panel styling
	panelStyle := styles.PanelStyle
	if p.focused {
		panelStyle = styles.FocusedPanelStyle
	}

	title := styles.RenderTitle(fmt.Sprintf("📊 Orderbook - %s", tickerName), p.focused)
	panel := lipgloss.JoinVertical(lipgloss.Left, title, content.String())

	return panelStyle.Width(p.width - 2).Height(p.height - 2).Render(panel)
}

// SetFocus sets the focus state of the panel.
func (p *OrderbookPanel) SetFocus(focused bool) {
	p.focused = focused
}

// SetSize sets the panel dimensions.
func (p *OrderbookPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetTicker sets the ticker to display.
func (p *OrderbookPanel) SetTicker(ticker market.Ticker) {
	p.ticker = ticker
	p.bids = nil
	p.asks = nil
	p.trades = nil
	p.scrollOffset = 0
}

// SetLevels sets the orderbook levels.
func (p *OrderbookPanel) SetLevels(bids, asks []orderbookview.Level) {
	p.bids = bids
	p.asks = asks
}

// SetTrades sets the recent trades.
func (p *OrderbookPanel) SetTrades(trades []core.TradeEvent) {
	p.trades = trades
}

// AddTrade adds a trade to the display.
func (p *OrderbookPanel) AddTrade(trade core.TradeEvent) {
	p.trades = append(p.trades, trade)
	// Keep only last 20 trades
	if len(p.trades) > 20 {
		p.trades = p.trades[len(p.trades)-20:]
	}
}

// Ticker returns the current ticker.
func (p *OrderbookPanel) Ticker() market.Ticker {
	return p.ticker
}
