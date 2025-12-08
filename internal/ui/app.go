package ui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UIModel represents the main Bubbletea model
type UIModel struct {
	layout           *Layout
	marketWidget     *MarketWidget
	newsWidget       *NewsWidget
	orderbookWidget  *OrderBookWidget
	orderentryWidget *OrderEntryWidget
	channels         *UIChannels
	termWidth        int
	termHeight       int
	lastUpdate       time.Time
	running          bool
}

// UIUpdateMsg represents UI update messages from channels
type UIUpdateMsg struct {
	event UIEvent
}

// TickMsg for regular UI refreshes
type TickMsg time.Time

// NewUIModel creates a new UI model
func NewUIModel(channels *UIChannels) *UIModel {
	layout := NewLayout()

	// Create widgets
	marketWidget := NewMarketWidget(channels)
	newsWidget := NewNewsWidget()
	orderbookWidget := NewOrderBookWidget()
	orderentryWidget := NewOrderEntryWidget()

	// Add widgets to layout
	layout.AddWidget(marketWidget)
	layout.AddWidget(orderbookWidget)
	layout.AddWidget(newsWidget)
	layout.AddWidget(orderentryWidget)

	return &UIModel{
		layout:           layout,
		marketWidget:     marketWidget,
		newsWidget:       newsWidget,
		orderbookWidget:  orderbookWidget,
		orderentryWidget: orderentryWidget,
		channels:         channels,
		termWidth:        80,
		termHeight:       24,
		lastUpdate:       time.Now(),
		running:          true,
	}
}

// Init initializes the Bubbletea model
func (m UIModel) Init() tea.Cmd {
	return tea.Batch(
		m.listenForEvents(),
		tickEvery(time.Millisecond*33), // ~30 FPS
	)
}

// Update handles Bubbletea updates
func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		return m, nil

	case UIUpdateMsg:
		// Update all widgets with the event
		m.layout.UpdateAll(msg.event)
		return m, m.listenForEvents() // Continue listening

	case TickMsg:
		// Regular UI refresh
		m.lastUpdate = time.Time(msg)
		return m, tickEvery(time.Millisecond * 33)
	}

	return m, nil
}

func (m UIModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.running = false
		close(m.channels.Shutdown)
		return m, tea.Quit

	case "tab":
		m.layout.NextFocus()
		return m, nil

	case "shift+tab":
		m.layout.PrevFocus()
		return m, nil

	default:
		// Pass key to focused widget
		widgets := m.layout.GetWidgets()
		for _, widget := range widgets {
			if widget.Focused() {
				if marketWidget, ok := widget.(*MarketWidget); ok {
					// Market widget handles arrow keys for stock selection
					marketWidget.HandleKey(msg.String())
				} else if orderWidget, ok := widget.(*OrderEntryWidget); ok {
					// Order entry widget handles all keys including arrows
					orderWidget.HandleKey(msg.String())
				}
				break
			}
		}
		return m, nil
	}
}

// View renders the UI
func (m UIModel) View() string {
	if !m.running {
		return ""
	}

	// Calculate layout dimensions with proper constraints
	statusBarHeight := 1
	availableHeight := m.termHeight - statusBarHeight
	halfWidth := m.termWidth / 2
	halfHeight := availableHeight / 2

	// Ensure minimum dimensions
	if halfWidth < 10 {
		halfWidth = 10
	}
	if halfHeight < 5 {
		halfHeight = 5
	}

	// Render widgets in a 2x2 grid with constrained dimensions
	topLeft := m.marketWidget.Render(halfWidth, halfHeight)
	topRight := m.orderbookWidget.Render(halfWidth, halfHeight)
	bottomLeft := m.newsWidget.Render(halfWidth, halfHeight)
	bottomRight := m.orderentryWidget.Render(halfWidth, halfHeight)

	// Combine into grid layout
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topLeft, topRight)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, bottomLeft, bottomRight)
	fullView := lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)

	// Add status bar
	statusBar := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("0")).
		Width(m.termWidth).
		Height(statusBarHeight).
		Align(lipgloss.Center).
		Render(fmt.Sprintf(" StockCraft Terminal | Last Update: %s | Press 'q' to quit ",
			m.lastUpdate.Format("15:04:05")))

	return lipgloss.JoinVertical(lipgloss.Left, fullView, statusBar)
}

// listenForEvents listens for UI events from channels
func (m UIModel) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		select {
		case event := <-m.channels.MarketUpdates:
			return UIUpdateMsg{event: event}
		case event := <-m.channels.OrderUpdates:
			return UIUpdateMsg{event: event}
		case event := <-m.channels.NewsUpdates:
			return UIUpdateMsg{event: event}
		case event := <-m.channels.StockSelections:
			return UIUpdateMsg{event: event}
		case <-m.channels.Shutdown:
			return tea.Quit()
		}
	}
}

// tickEvery returns a command that sends a TickMsg at regular intervals
func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// RunUI starts the Bubbletea application
func RunUI(ctx context.Context, channels *UIChannels) error {
	m := NewUIModel(channels)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	_, err := p.Run()
	return err
}
