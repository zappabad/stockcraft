package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zappabad/stockcraft/internal/engine"
)

// OrderBookWidget displays recent orders and trades
type OrderBookWidget struct {
	BaseWidget
	recentOrders []engine.Order
	lastTick     int
	maxOrders    int
	filterSymbol string
}

func NewOrderBookWidget() *OrderBookWidget {
	return &OrderBookWidget{
		BaseWidget:   NewBaseWidget(),
		recentOrders: make([]engine.Order, 0),
		lastTick:     0,
		maxOrders:    15,    // Keep last 15 orders
		filterSymbol: "FOO", // Default to FOO
	}
}

func (w *OrderBookWidget) Update(event UIEvent) bool {
	if orderEvent, ok := event.(OrderUpdateEvent); ok {
		// Add new orders
		w.recentOrders = append(w.recentOrders, orderEvent.Orders...)

		// Keep only last maxOrders
		if len(w.recentOrders) > w.maxOrders {
			w.recentOrders = w.recentOrders[len(w.recentOrders)-w.maxOrders:]
		}

		w.lastTick = orderEvent.Tick
		return true
	} else if stockEvent, ok := event.(StockSelectionEvent); ok {
		// Update filter symbol when stock selection changes
		w.filterSymbol = stockEvent.Symbol
		return true
	}
	return false
}

func (w *OrderBookWidget) Render(width, height int) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	if w.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("32"))
	}

	// Build header
	header := fmt.Sprintf("Order Flow for %s (Tick: %d)", w.filterSymbol, w.lastTick)

	// Build order display
	var lines []string
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", len(header)))

	// Filter orders for selected symbol
	var filteredOrders []engine.Order
	for _, order := range w.recentOrders {
		if order.Symbol == w.filterSymbol {
			filteredOrders = append(filteredOrders, order)
		}
	}

	if len(filteredOrders) == 0 {
		lines = append(lines, fmt.Sprintf("No orders for %s", w.filterSymbol))
	} else {
		// Column headers
		lines = append(lines, fmt.Sprintf("%-8s %-4s %-4s %6s %8s", "Trader", "Symb", "Side", "Qty", "Price"))
		lines = append(lines, strings.Repeat("─", 35))

		// Calculate available lines for orders (subtract header + separator + column headers + border)
		availableLines := height - 6 // 2 for border, 2 for header+separator, 2 for column headers
		if availableLines < 1 {
			availableLines = 1
		}

		// Show recent filtered orders (newest first), but only what fits
		displayCount := len(filteredOrders)
		if displayCount > availableLines {
			displayCount = availableLines
		}

		for i := len(filteredOrders) - 1; i >= len(filteredOrders)-displayCount; i-- {
			order := filteredOrders[i]

			// Truncate trader ID for display
			traderDisplay := order.TraderID
			if len(traderDisplay) > 8 {
				traderDisplay = traderDisplay[:8]
			}

			// Format side
			sideStr := "BUY"
			sideColor := lipgloss.Color("82") // Green for buy
			if order.Side == engine.SideSell {
				sideStr = "SELL"
				sideColor = lipgloss.Color("196") // Red for sell
			}

			sideStyled := lipgloss.NewStyle().Foreground(sideColor).Render(sideStr)

			orderLine := fmt.Sprintf("%-8s %-4s %-4s %6d %8.2f",
				traderDisplay,
				order.Symbol,
				sideStyled,
				order.Quantity,
				order.Price,
			)

			lines = append(lines, orderLine)
		}
	}

	content := strings.Join(lines, "\n")

	// Apply border and sizing
	contentWidth := width - 2   // Account for border
	contentHeight := height - 2 // Account for border

	return borderStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)
}
