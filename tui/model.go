package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zappabad/stockcraft/internal/market"
	marketservice "github.com/zappabad/stockcraft/internal/market/service"
	marketview "github.com/zappabad/stockcraft/internal/market/view"
	newsservice "github.com/zappabad/stockcraft/internal/news/service"
	newsview "github.com/zappabad/stockcraft/internal/news/view"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
	"github.com/zappabad/stockcraft/tui/panels"
	"github.com/zappabad/stockcraft/tui/styles"
)

// PanelFocus represents which panel is currently focused.
type PanelFocus int

const (
	FocusMarket     PanelFocus = 0
	FocusOrderbook  PanelFocus = 1
	FocusChart      PanelFocus = 2
	FocusNews       PanelFocus = 3
	FocusOrderInput PanelFocus = 4
)

// Model is the main TUI application model.
type Model struct {
	// Services
	marketService *marketservice.MarketService
	newsService   *newsservice.NewsService

	// Tickers
	tickers   []market.Ticker
	tickerMap map[market.TickerID]market.Ticker

	// User ID for placing orders
	userID core.UserID

	// Panels
	marketPanel     *panels.MarketOverviewPanel
	orderbookPanel  *panels.OrderbookPanel
	newsPanel       *panels.NewsPanel
	orderInputPanel *panels.OrderInputPanel
	chartPanel      *panels.CandlestickPanel

	// Focus management
	focusedPanel PanelFocus

	// Window dimensions
	width  int
	height int

	// Status
	statusMsg string
	ready     bool
}

// NewModel creates a new TUI model.
func NewModel(marketService *marketservice.MarketService, newsService *newsservice.NewsService, userID core.UserID) *Model {
	tickers := marketService.GetTickers()

	// Build ticker map
	tickerMap := make(map[market.TickerID]market.Ticker)
	for _, t := range tickers {
		tickerMap[t.TickerID()] = t
	}

	// Create panels
	marketPanel := panels.NewMarketOverviewPanel(tickers)
	orderbookPanel := panels.NewOrderbookPanel()
	newsPanel := panels.NewNewsPanel()
	orderInputPanel := panels.NewOrderInputPanel(tickers)
	chartPanel := panels.NewCandlestickPanel()

	// Set initial ticker
	if len(tickers) > 0 {
		orderbookPanel.SetTicker(tickers[0])
		chartPanel.SetTicker(tickers[0])
	}

	return &Model{
		marketService:   marketService,
		newsService:     newsService,
		tickers:         tickers,
		tickerMap:       tickerMap,
		userID:          userID,
		marketPanel:     marketPanel,
		orderbookPanel:  orderbookPanel,
		newsPanel:       newsPanel,
		orderInputPanel: orderInputPanel,
		chartPanel:      chartPanel,
		focusedPanel:    FocusOrderInput,
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.marketPanel.Init(),
		m.orderbookPanel.Init(),
		m.newsPanel.Init(),
		m.orderInputPanel.Init(),
		m.chartPanel.Init(),
		m.listenMarketEvents(),
		m.listenNewsEvents(),
		m.tickRefresh(),
	)
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		// Cycle focus with tab
		case "tab":
			m.cycleFocus()

		// Reverse cycle focus with shift+tab
		case "shift+tab":
			m.focusedPanel--
			if m.focusedPanel < 0 {
				m.focusedPanel = 4
			}

		// Direct panel focus with F1-F5
		case "f1":
			m.setFocus(FocusMarket)
		case "f2":
			m.setFocus(FocusOrderbook)
		case "f3":
			m.setFocus(FocusNews)
		case "f4":
			m.setFocus(FocusOrderInput)
		case "f5":
			m.setFocus(FocusChart)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updatePanelSizes()
		m.ready = true

	case panels.MarketUpdateMsg:
		m.handleMarketUpdate(msg)

	case panels.NewsUpdateMsg:
		m.newsPanel.AddNews(msg.Item)

	case panels.TickerSelectedMsg:
		m.orderbookPanel.SetTicker(msg.Ticker)
		m.chartPanel.SetTicker(msg.Ticker)
		m.updateOrderbookData()

	case panels.OrderSubmitMsg:
		cmds = append(cmds, m.submitOrder(msg))

	case orderResultMsg:
		m.statusMsg = msg.message

	case tickMsg:
		m.updateAllData()
		cmds = append(cmds, m.tickRefresh())
	}

	// Update focused panel
	m.updateFocusedPanel(msg, &cmds)

	return m, tea.Batch(cmds...)
}

func (m *Model) updateFocusedPanel(msg tea.Msg, cmds *[]tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedPanel {
	case FocusMarket:
		m.marketPanel, cmd = m.marketPanel.Update(msg)
		// Check if selection changed
		selected := m.marketPanel.SelectedTicker()
		if selected.Name != "" && selected.Name != m.orderbookPanel.Ticker().Name {
			m.orderbookPanel.SetTicker(selected)
			m.chartPanel.SetTicker(selected)
			m.updateOrderbookData()
		}
	case FocusOrderbook:
		m.orderbookPanel, cmd = m.orderbookPanel.Update(msg)
	case FocusNews:
		m.newsPanel, cmd = m.newsPanel.Update(msg)
	case FocusOrderInput:
		m.orderInputPanel, cmd = m.orderInputPanel.Update(msg)
	case FocusChart:
		m.chartPanel, cmd = m.chartPanel.Update(msg)
	}

	if cmd != nil {
		*cmds = append(*cmds, cmd)
	}
}

// View renders the UI.
func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Update focus states
	m.marketPanel.SetFocus(m.focusedPanel == FocusMarket)
	m.orderbookPanel.SetFocus(m.focusedPanel == FocusOrderbook)
	m.newsPanel.SetFocus(m.focusedPanel == FocusNews)
	m.orderInputPanel.SetFocus(m.focusedPanel == FocusOrderInput)
	m.chartPanel.SetFocus(m.focusedPanel == FocusChart)

	// Layout:
	// ┌─────────────────────────────────────────────┐
	// │  Market Overview  │  Orderbook  │   Chart   │
	// │                   │             │           │
	// ├───────────────────┼─────────────┴───────────┤
	// │      News         │      Order Input        │
	// └───────────────────┴─────────────────────────┘

	// Calculate column widths
	leftWidth := m.width / 3
	middleWidth := m.width / 3
	rightWidth := m.width - leftWidth - middleWidth

	// Calculate row heights
	topHeight := (m.height - 3) * 2 / 3 // 2/3 for top row
	bottomHeight := m.height - topHeight - 3

	// Render top row panels
	m.marketPanel.SetSize(leftWidth, topHeight)
	m.orderbookPanel.SetSize(middleWidth, topHeight)
	m.chartPanel.SetSize(rightWidth, topHeight)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.marketPanel.View(),
		m.orderbookPanel.View(),
		m.chartPanel.View(),
	)

	// Render bottom row panels
	m.newsPanel.SetSize(leftWidth, bottomHeight)
	m.orderInputPanel.SetSize(m.width-leftWidth, bottomHeight)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.newsPanel.View(),
		m.orderInputPanel.View(),
	)

	// Status bar
	statusBar := m.renderStatusBar()

	// Join all rows
	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow, statusBar)
}

func (m *Model) renderStatusBar() string {
	// Help text
	help := []string{
		styles.StatusBarKeyStyle.Render("F1-F5") + styles.StatusBarDescStyle.Render(" panels"),
		styles.StatusBarKeyStyle.Render("Tab/Enter") + styles.StatusBarDescStyle.Render(" navigate"),
		styles.StatusBarKeyStyle.Render("↑↓") + styles.StatusBarDescStyle.Render(" select"),
		styles.StatusBarKeyStyle.Render("q") + styles.StatusBarDescStyle.Render(" quit"),
	}

	helpStr := lipgloss.JoinHorizontal(lipgloss.Center, help[0], " │ ", help[1], " │ ", help[2], " │ ", help[3])

	// Status message
	status := ""
	if m.statusMsg != "" {
		status = " │ " + m.statusMsg
	}

	return styles.StatusBarStyle.Width(m.width).Render(helpStr + status)
}

func (m *Model) setFocus(panel PanelFocus) {
	m.focusedPanel = panel
}

func (m *Model) cycleFocus() {
	m.focusedPanel = (m.focusedPanel + 1) % 5
}

func (m *Model) updatePanelSizes() {
	// Will be updated in View()
}

func (m *Model) handleMarketUpdate(msg panels.MarketUpdateMsg) {
	// Update market overview
	snap := m.marketService.Snapshot()
	m.marketPanel.SetSnapshot(snap)

	// If this is for the currently selected ticker, update orderbook
	if ticker, ok := m.tickerMap[msg.Ticker]; ok {
		if ticker.Name == m.orderbookPanel.Ticker().Name {
			// Update orderbook
			bids, _ := m.marketService.GetLevels(msg.Ticker, core.SideBuy)
			asks, _ := m.marketService.GetLevels(msg.Ticker, core.SideSell)
			m.orderbookPanel.SetLevels(bids, asks)

			// Handle trade events for chart
			if trade, ok := msg.Event.(core.TradeEvent); ok {
				m.orderbookPanel.AddTrade(trade)
				m.chartPanel.AddTrade(trade)
			}
		}
	}
}

func (m *Model) updateAllData() {
	// Update market snapshot
	snap := m.marketService.Snapshot()
	m.marketPanel.SetSnapshot(snap)

	// Update orderbook
	m.updateOrderbookData()

	// Update news
	news := m.newsService.Latest(20)
	m.newsPanel.SetNews(news)
}

func (m *Model) updateOrderbookData() {
	ticker := m.orderbookPanel.Ticker()
	if ticker.Name == "" {
		return
	}

	tid := ticker.TickerID()
	bids, _ := m.marketService.GetLevels(tid, core.SideBuy)
	asks, _ := m.marketService.GetLevels(tid, core.SideSell)
	m.orderbookPanel.SetLevels(bids, asks)

	trades, _ := m.marketService.GetTradesLast(tid, 20)
	m.orderbookPanel.SetTrades(trades)

	// Also populate chart with historical trades
	for _, trade := range trades {
		m.chartPanel.AddTrade(trade)
	}
}

func (m *Model) submitOrder(order panels.OrderSubmitMsg) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tid := order.Ticker.TickerID()

		var err error
		var report core.SubmitReport

		if order.OrderKind == core.OrderKindLimit {
			report, err = m.marketService.SubmitLimit(ctx, tid, m.userID, order.Side, order.Price, order.Quantity)
		} else {
			report, err = m.marketService.SubmitMarket(ctx, tid, m.userID, order.Side, order.Quantity)
		}

		if err != nil {
			return orderResultMsg{message: "❌ Order failed: " + err.Error()}
		}

		// Calculate filled amount from fills
		var filled core.Size
		var totalValue int64
		for _, fill := range report.Fills {
			filled += fill.Size
			totalValue += int64(fill.Price) * int64(fill.Size)
		}

		if filled > 0 {
			avgPrice := totalValue / int64(filled)
			return orderResultMsg{message: fmt.Sprintf("✓ Filled %d @ %d", filled, avgPrice)}
		}
		return orderResultMsg{message: fmt.Sprintf("✓ Order placed (ID: %d)", report.OrderID)}
	}
}

func (m *Model) listenMarketEvents() tea.Cmd {
	return func() tea.Msg {
		events := m.marketService.Events()
		ev, ok := <-events
		if !ok {
			return nil
		}
		return panels.MarketUpdateMsg{
			Ticker: ev.Ticker,
			Event:  ev.Event,
		}
	}
}

func (m *Model) listenNewsEvents() tea.Cmd {
	return func() tea.Msg {
		events := m.newsService.Events()
		ev, ok := <-events
		if !ok {
			return nil
		}
		return panels.NewsUpdateMsg{Item: ev.Item}
	}
}

// tickMsg is sent periodically to refresh data.
type tickMsg struct{}

func (m *Model) tickRefresh() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// orderResultMsg is sent after an order is processed.
type orderResultMsg struct {
	message string
}

// Msg types from services for re-export
type MarketEventMsg = marketview.MarketEvent
type NewsEventMsg = newsview.NewsEvent
