package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zappabad/stockcraft/internal/engine"
)

type OrderField int

const (
	FieldSymbol OrderField = iota
	FieldSide
	FieldQuantity
	FieldPrice
)

// OrderEntryWidget provides a form for entering orders
type OrderEntryWidget struct {
	BaseWidget
	currentField OrderField
	symbol       string
	side         engine.Side
	quantity     string
	price        string
	statusMsg    string
}

func NewOrderEntryWidget() *OrderEntryWidget {
	return &OrderEntryWidget{
		BaseWidget:   NewBaseWidget(),
		currentField: FieldSymbol,
		symbol:       "FOO", // Default symbol
		side:         engine.SideBuy,
		quantity:     "100",
		price:        "100.00",
		statusMsg:    "Ready",
	}
}

func (w *OrderEntryWidget) Update(event UIEvent) bool {
	// Handle stock selection events to auto-populate symbol field
	if stockEvent, ok := event.(StockSelectionEvent); ok {
		w.symbol = stockEvent.Symbol
		return true
	}
	return false
}

// HandleKey processes keyboard input for order entry
func (w *OrderEntryWidget) HandleKey(key string) {
	switch key {
	case "tab":
		w.nextField()
	case "shift+tab":
		w.prevField()
	case "down", "right":
		w.nextField()
	case "up", "left":
		w.prevField()
	case "enter":
		w.submitOrder()
	default:
		w.handleFieldInput(key)
	}
}

func (w *OrderEntryWidget) nextField() {
	w.currentField = (w.currentField + 1) % 4
}

func (w *OrderEntryWidget) prevField() {
	if w.currentField == 0 {
		w.currentField = 3
	} else {
		w.currentField--
	}
}

func (w *OrderEntryWidget) handleFieldInput(key string) {
	switch w.currentField {
	case FieldSymbol:
		w.handleSymbolInput(key)
	case FieldSide:
		w.handleSideInput(key)
	case FieldQuantity:
		w.handleNumericInput(key, &w.quantity, 8)
	case FieldPrice:
		w.handleNumericInput(key, &w.price, 10)
	}
}

func (w *OrderEntryWidget) handleSymbolInput(key string) {
	if key == "backspace" {
		if len(w.symbol) > 0 {
			w.symbol = w.symbol[:len(w.symbol)-1]
		}
	} else if len(key) == 1 && len(w.symbol) < 10 {
		// Only allow alphanumeric characters
		c := key[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			w.symbol += strings.ToUpper(key)
		}
	}
}

func (w *OrderEntryWidget) handleSideInput(key string) {
	if key == " " || key == "b" || key == "s" {
		if w.side == engine.SideBuy {
			w.side = engine.SideSell
		} else {
			w.side = engine.SideBuy
		}
	}
}

func (w *OrderEntryWidget) handleNumericInput(key string, field *string, maxLen int) {
	if key == "backspace" {
		if len(*field) > 0 {
			*field = (*field)[:len(*field)-1]
		}
	} else if len(key) == 1 && len(*field) < maxLen {
		// Allow digits and decimal point
		c := key[0]
		if (c >= '0' && c <= '9') || (c == '.' && !strings.Contains(*field, ".")) {
			*field += key
		}
	}
}

func (w *OrderEntryWidget) submitOrder() {
	// Validate inputs
	if w.symbol == "" {
		w.statusMsg = "Error: Symbol required"
		return
	}

	qty, err := strconv.Atoi(w.quantity)
	if err != nil || qty <= 0 {
		w.statusMsg = "Error: Invalid quantity"
		return
	}

	price, err := strconv.ParseFloat(w.price, 64)
	if err != nil || price <= 0 {
		w.statusMsg = "Error: Invalid price"
		return
	}

	// For now, just show a success message (not actually submitting to market)
	sideStr := "BUY"
	if w.side == engine.SideSell {
		sideStr = "SELL"
	}
	w.statusMsg = fmt.Sprintf("Order: %s %d %s @ %.2f (placeholder)", sideStr, qty, w.symbol, price)
}

func (w *OrderEntryWidget) Render(width, height int) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	if w.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("32"))
	}

	// Field highlighting
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("32")).
		Foreground(lipgloss.Color("0"))

	normalStyle := lipgloss.NewStyle()

	// Build header
	header := "Order Entry"

	// Build form
	var lines []string
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", len(header)))
	lines = append(lines, "")

	// Symbol field
	symbolStyle := normalStyle
	if w.currentField == FieldSymbol {
		symbolStyle = highlightStyle
	}
	symbolValue := w.symbol
	if symbolValue == "" {
		symbolValue = "_"
	}
	lines = append(lines, fmt.Sprintf("Symbol: %s", symbolStyle.Render(fmt.Sprintf(" %-10s ", symbolValue))))

	// Side field
	sideStyle := normalStyle
	if w.currentField == FieldSide {
		sideStyle = highlightStyle
	}
	sideValue := "BUY"
	if w.side == engine.SideSell {
		sideValue = "SELL"
	}
	lines = append(lines, fmt.Sprintf("Side:   %s", sideStyle.Render(fmt.Sprintf(" %-4s ", sideValue))))

	// Quantity field
	qtyStyle := normalStyle
	if w.currentField == FieldQuantity {
		qtyStyle = highlightStyle
	}
	qtyValue := w.quantity
	if qtyValue == "" {
		qtyValue = "_"
	}
	lines = append(lines, fmt.Sprintf("Qty:    %s", qtyStyle.Render(fmt.Sprintf(" %-8s ", qtyValue))))

	// Price field
	priceStyle := normalStyle
	if w.currentField == FieldPrice {
		priceStyle = highlightStyle
	}
	priceValue := w.price
	if priceValue == "" {
		priceValue = "_"
	}
	lines = append(lines, fmt.Sprintf("Price:  %s", priceStyle.Render(fmt.Sprintf(" %-10s ", priceValue))))

	// Instructions
	lines = append(lines, "")
	lines = append(lines, "Tab: Next field | Enter: Submit")
	lines = append(lines, "Space: Toggle side (B/S)")

	// Status
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Status: %s", w.statusMsg))

	content := strings.Join(lines, "\n")

	// Apply border and sizing
	contentWidth := width - 2   // Account for border
	contentHeight := height - 2 // Account for border

	return borderStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)
}
