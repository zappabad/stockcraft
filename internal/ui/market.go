package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// MarketWidget displays live stock prices with color-coded changes
type MarketWidget struct {
	BaseWidget
	prices        map[string]float64
	prevPrices    map[string]float64
	changes       map[string]float64
	lastTick      int
	selectedStock string
	channels      *UIChannels // for publishing selection changes
}

func NewMarketWidget(channels *UIChannels) *MarketWidget {
	return &MarketWidget{
		BaseWidget:    NewBaseWidget(),
		prices:        make(map[string]float64),
		prevPrices:    make(map[string]float64),
		changes:       make(map[string]float64),
		lastTick:      0,
		selectedStock: "FOO", // Default selection
		channels:      channels,
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

// HandleKey processes keyboard input for stock navigation
func (w *MarketWidget) HandleKey(key string) {
	if !w.focused {
		return
	}

	// Get available symbols in order
	orderedSymbols := []string{"FOO", "BAR", "AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "BRK.A", "NVDA", "JPM", "V", "JNJ", "WMT", "PG", "DIS"}

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

	allSymbols := append(orderedSymbols, otherSymbols...)

	// Find current selection index
	currentIndex := -1
	for i, symbol := range allSymbols {
		if symbol == w.selectedStock {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 && len(allSymbols) > 0 {
		currentIndex = 0
		w.selectedStock = allSymbols[0]
	}

	switch key {
	case "up":
		if currentIndex > 0 {
			w.selectedStock = allSymbols[currentIndex-1]
			w.publishSelection()
		}
	case "down":
		if currentIndex < len(allSymbols)-1 {
			w.selectedStock = allSymbols[currentIndex+1]
			w.publishSelection()
		}
	}
}

// publishSelection sends the current selection to other widgets
func (w *MarketWidget) publishSelection() {
	if w.channels != nil {
		select {
		case w.channels.StockSelections <- StockSelectionEvent{Symbol: w.selectedStock}:
			// Selection sent
		default:
			// Channel full, drop event
		}
	}
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
		orderedSymbols := []string{"FOO", "BAR", "AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "BRK.A", "NVDA", "JPM", "V", "JNJ", "WMT", "PG", "DIS"}

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

			// Apply selection highlighting
			line := fmt.Sprintf("%s %s", priceStr, changeDisplay)
			if symbol == w.selectedStock {
				// Gray background for selected stock
				selectionStyle := lipgloss.NewStyle().Background(lipgloss.Color("240"))
				line = selectionStyle.Render(line)
			}

			lines = append(lines, line)
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
