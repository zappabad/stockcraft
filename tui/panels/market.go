package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zappabad/stockcraft/internal/market"
	marketview "github.com/zappabad/stockcraft/internal/market/view"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
	"github.com/zappabad/stockcraft/tui/styles"
)

// MarketOverviewPanel displays current prices for all tickers.
type MarketOverviewPanel struct {
	tickers       []market.Ticker
	tickerPrices  map[market.TickerID]marketview.BestPrices
	selectedIndex int
	focused       bool
	width         int
	height        int
}

// NewMarketOverviewPanel creates a new market overview panel.
func NewMarketOverviewPanel(tickers []market.Ticker) *MarketOverviewPanel {
	return &MarketOverviewPanel{
		tickers:      tickers,
		tickerPrices: make(map[market.TickerID]marketview.BestPrices),
	}
}

// Init initializes the panel.
func (p *MarketOverviewPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the panel.
func (p *MarketOverviewPanel) Update(msg tea.Msg) (*MarketOverviewPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if p.selectedIndex > 0 {
				p.selectedIndex--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if p.selectedIndex < len(p.tickers)-1 {
				p.selectedIndex++
			}
		}
	}
	return p, nil
}

// View renders the panel.
func (p *MarketOverviewPanel) View() string {
	var content strings.Builder

	// Header
	header := fmt.Sprintf("%-8s %10s %10s %10s %10s",
		"Ticker", "Bid", "BidSz", "Ask", "AskSz")
	content.WriteString(styles.HeaderStyle.Render(header))
	content.WriteString("\n")

	// Rows
	for i, ticker := range p.tickers {
		tid := ticker.TickerID()
		prices := p.tickerPrices[tid]

		bidPrice := "-"
		bidSize := "-"
		askPrice := "-"
		askSize := "-"

		if prices.BidOK {
			bidPrice = formatPrice(int64(prices.BidPrice), ticker.Decimals)
			bidSize = fmt.Sprintf("%d", prices.BidSize)
		}
		if prices.AskOK {
			askPrice = formatPrice(int64(prices.AskPrice), ticker.Decimals)
			askSize = fmt.Sprintf("%d", prices.AskSize)
		}

		row := fmt.Sprintf("%-8s %10s %10s %10s %10s",
			ticker.Name, bidPrice, bidSize, askPrice, askSize)

		style := styles.RowStyle
		if i == p.selectedIndex && p.focused {
			style = styles.SelectedRowStyle
		}
		content.WriteString(style.Render(row))
		if i < len(p.tickers)-1 {
			content.WriteString("\n")
		}
	}

	// Apply panel styling
	panelStyle := styles.PanelStyle
	if p.focused {
		panelStyle = styles.FocusedPanelStyle
	}

	title := styles.RenderTitle("📈 Market Overview", p.focused)
	panel := lipgloss.JoinVertical(lipgloss.Left, title, content.String())

	return panelStyle.Width(p.width - 2).Height(p.height - 2).Render(panel)
}

// SetFocus sets the focus state of the panel.
func (p *MarketOverviewPanel) SetFocus(focused bool) {
	p.focused = focused
}

// SetSize sets the panel dimensions.
func (p *MarketOverviewPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// UpdatePrices updates the prices for a given ticker.
func (p *MarketOverviewPanel) UpdatePrices(tid market.TickerID, prices marketview.BestPrices) {
	p.tickerPrices[tid] = prices
}

// SetSnapshot sets all ticker prices from a market snapshot.
func (p *MarketOverviewPanel) SetSnapshot(snap marketview.MarketSnapshot) {
	for tid, prices := range snap.ByTicker {
		p.tickerPrices[tid] = prices
	}
}

// SelectedTicker returns the currently selected ticker.
func (p *MarketOverviewPanel) SelectedTicker() market.Ticker {
	if p.selectedIndex >= 0 && p.selectedIndex < len(p.tickers) {
		return p.tickers[p.selectedIndex]
	}
	return market.Ticker{}
}

// Helper function to format price
func formatPrice(price int64, decimals int8) string {
	if decimals <= 0 {
		return fmt.Sprintf("%d", price)
	}
	divisor := int64(1)
	for i := int8(0); i < decimals; i++ {
		divisor *= 10
	}
	whole := price / divisor
	frac := price % divisor
	if frac < 0 {
		frac = -frac
	}
	formatStr := fmt.Sprintf("%%d.%%0%dd", decimals)
	return fmt.Sprintf(formatStr, whole, frac)
}

// TickerSelectedMsg is sent when a ticker is selected.
type TickerSelectedMsg struct {
	Ticker market.Ticker
}

// MarketUpdateMsg is sent when market data updates.
type MarketUpdateMsg struct {
	Ticker market.TickerID
	Event  core.Event
}
