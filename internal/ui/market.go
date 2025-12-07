package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// MarketWidget displays live stock prices with color-coded changes
type MarketWidget struct {
	BaseWidget
	prices     map[string]float64
	prevPrices map[string]float64
	changes    map[string]float64
	lastTick   int
}

func NewMarketWidget() *MarketWidget {
	return &MarketWidget{
		BaseWidget: NewBaseWidget(),
		prices:     make(map[string]float64),
		prevPrices: make(map[string]float64),
		changes:    make(map[string]float64),
		lastTick:   0,
	}
}

func (w *MarketWidget) Update(event UIEvent) bool {
	if marketEvent, ok := event.(MarketUpdateEvent); ok {
		// Store previous prices for change calculation
		for symbol, price := range w.prices {
			w.prevPrices[symbol] = price
		}

		// Update current prices
		w.prices = make(map[string]float64)
		for symbol, price := range marketEvent.Prices {
			w.prices[symbol] = price
		}

		// Calculate changes
		w.changes = make(map[string]float64)
		for symbol, price := range w.prices {
			if prevPrice, exists := w.prevPrices[symbol]; exists {
				w.changes[symbol] = price - prevPrice
			} else {
				w.changes[symbol] = 0
			}
		}

		w.lastTick = marketEvent.Tick
		return true
	}
	return false
}

func (w *MarketWidget) Render(width, height int) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	if w.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("32"))
	}

	// Build header
	header := fmt.Sprintf("Market Data (Tick: %d)", w.lastTick)

	// Build price display
	var lines []string
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", len(header)))

	if len(w.prices) == 0 {
		lines = append(lines, "No market data available")
	} else {
		// Use deterministic order - prioritize FOO, then BAR, then alphabetical for any others
		orderedSymbols := []string{"FOO", "BAR"}

		// Add any other symbols in alphabetical order
		var otherSymbols []string
		for symbol := range w.prices {
			found := false
			for _, known := range orderedSymbols {
				if symbol == known {
					found = true
					break
				}
			}
			if !found {
				otherSymbols = append(otherSymbols, symbol)
			}
		}

		// Sort other symbols alphabetically
		for i := 0; i < len(otherSymbols); i++ {
			for j := i + 1; j < len(otherSymbols); j++ {
				if otherSymbols[i] > otherSymbols[j] {
					otherSymbols[i], otherSymbols[j] = otherSymbols[j], otherSymbols[i]
				}
			}
		}

		// Combine ordered symbols with other symbols
		finalOrder := append(orderedSymbols, otherSymbols...)

		// Display symbols in deterministic order
		for _, symbol := range finalOrder {
			price, exists := w.prices[symbol]
			if !exists {
				continue // Skip symbols that don't have prices
			}

			change := w.changes[symbol]
			changeStr := ""
			colorStyle := lipgloss.NewStyle()

			if change > 0 {
				changeStr = fmt.Sprintf("+%.2f", change)
				colorStyle = colorStyle.Foreground(lipgloss.Color("82")) // Green
			} else if change < 0 {
				changeStr = fmt.Sprintf("%.2f", change)
				colorStyle = colorStyle.Foreground(lipgloss.Color("196")) // Red
			} else {
				changeStr = " 0.00"
				colorStyle = colorStyle.Foreground(lipgloss.Color("245")) // Gray
			}

			priceStr := fmt.Sprintf("%-4s $%8.2f", symbol, price)
			changeDisplay := colorStyle.Render(fmt.Sprintf("(%s)", changeStr))

			lines = append(lines, fmt.Sprintf("%s %s", priceStr, changeDisplay))
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
