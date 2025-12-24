package panels

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zappabad/stockcraft/internal/market"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
	"github.com/zappabad/stockcraft/tui/styles"
)

// OrderInputField represents the currently focused input field.
type OrderInputField int

const (
	FieldTicker OrderInputField = iota
	FieldSide
	FieldType
	FieldPrice
	FieldQuantity
	FieldSubmit
)

// OrderInputPanel handles order input with autocomplete.
type OrderInputPanel struct {
	tickers       []market.Ticker
	tickerInput   textinput.Model
	priceInput    textinput.Model
	quantityInput textinput.Model

	// Dropdown state
	showDropdown     bool
	dropdownItems    []string
	dropdownFiltered []string
	dropdownIndex    int

	// Side dropdown
	sideOptions []string
	sideIndex   int

	// Order type dropdown
	typeOptions []string
	typeIndex   int

	// Current field
	currentField OrderInputField

	// Selected values
	selectedTicker *market.Ticker

	focused bool
	width   int
	height  int
}

// NewOrderInputPanel creates a new order input panel.
func NewOrderInputPanel(tickers []market.Ticker) *OrderInputPanel {
	// Create ticker dropdown items
	tickerNames := make([]string, len(tickers))
	for i, t := range tickers {
		tickerNames[i] = t.Name
	}

	// Create text inputs
	tickerInput := textinput.New()
	tickerInput.Placeholder = "Search ticker..."
	tickerInput.Width = 15
	tickerInput.CharLimit = 10

	priceInput := textinput.New()
	priceInput.Placeholder = "Price"
	priceInput.Width = 10
	priceInput.CharLimit = 15

	quantityInput := textinput.New()
	quantityInput.Placeholder = "Quantity"
	quantityInput.Width = 10
	quantityInput.CharLimit = 15

	return &OrderInputPanel{
		tickers:          tickers,
		tickerInput:      tickerInput,
		priceInput:       priceInput,
		quantityInput:    quantityInput,
		dropdownItems:    tickerNames,
		dropdownFiltered: tickerNames,
		sideOptions:      []string{"BUY", "SELL"},
		typeOptions:      []string{"LIMIT", "MARKET"},
		currentField:     FieldTicker,
	}
}

// Init initializes the panel.
func (p *OrderInputPanel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the panel.
func (p *OrderInputPanel) Update(msg tea.Msg) (*OrderInputPanel, tea.Cmd) {
	if !p.focused {
		return p, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		// down arrow to next field
		case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
			p.nextField()
			return p, nil

		// up arrow to previous field
		case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
			p.prevField()
			return p, nil

		// Enter to submit or select dropdown
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if p.currentField == FieldSubmit {
				return p, p.submitOrder()
			}
			if p.showDropdown && p.currentField == FieldTicker {
				p.selectDropdownItem()
				p.showDropdown = false
				p.nextField()
				return p, nil
			}
			p.nextField()
			return p, nil

		// Escape to close dropdown
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			p.showDropdown = false
			return p, nil

		// Arrow keys for dropdown navigation
		case key.Matches(msg, key.NewBinding(key.WithKeys("left"))):
			if p.showDropdown {
				if p.dropdownIndex > 0 {
					p.dropdownIndex--
				}
				return p, nil
			}
			if p.currentField == FieldSide {
				if p.sideIndex > 0 {
					p.sideIndex--
				}
				return p, nil
			}
			if p.currentField == FieldType {
				if p.typeIndex > 0 {
					p.typeIndex--
				}
				return p, nil
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("right"))):
			if p.showDropdown {
				if p.dropdownIndex < len(p.dropdownFiltered)-1 {
					p.dropdownIndex++
				}
				return p, nil
			}
			if p.currentField == FieldSide {
				if p.sideIndex < len(p.sideOptions)-1 {
					p.sideIndex++
				}
				return p, nil
			}
			if p.currentField == FieldType {
				if p.typeIndex < len(p.typeOptions)-1 {
					p.typeIndex++
				}
				return p, nil
			}
		}
	}

	// Update the appropriate text input
	switch p.currentField {
	case FieldTicker:
		p.tickerInput, cmd = p.tickerInput.Update(msg)
		p.filterDropdown(p.tickerInput.Value())
		p.showDropdown = len(p.tickerInput.Value()) > 0

	case FieldPrice:
		p.priceInput, cmd = p.priceInput.Update(msg)

	case FieldQuantity:
		p.quantityInput, cmd = p.quantityInput.Update(msg)
	}

	return p, cmd
}

// View renders the panel.
func (p *OrderInputPanel) View() string {
	var content strings.Builder

	// Ticker field with dropdown
	content.WriteString(p.renderField("Ticker\n", FieldTicker, p.renderTickerField()))
	content.WriteString("\n")

	// Side field
	content.WriteString(p.renderField("Side", FieldSide, p.renderSideField()))
	content.WriteString("\n")

	// Type field
	content.WriteString(p.renderField("Type", FieldType, p.renderTypeField()))
	content.WriteString("\n")

	// Price field (only show for limit orders)
	if p.typeIndex == 0 { // LIMIT
		content.WriteString(p.renderField("Price", FieldPrice, p.priceInput.View()))
		content.WriteString("\n")
	}

	// Quantity field
	content.WriteString(p.renderField("Qty", FieldQuantity, p.quantityInput.View()))
	content.WriteString("\n\n")

	// Submit button
	submitStyle := styles.InputStyle
	if p.currentField == FieldSubmit && p.focused {
		submitStyle = styles.FocusedInputStyle.Bold(true).Foreground(styles.PrimaryColor)
	}
	content.WriteString(submitStyle.Render("  [Submit Order]  "))

	// Order summary
	content.WriteString("\n\n")
	content.WriteString(p.renderOrderSummary())

	// Apply panel styling
	panelStyle := styles.PanelStyle
	if p.focused {
		panelStyle = styles.FocusedPanelStyle
	}

	title := styles.RenderTitle("📝 Order Entry", p.focused)
	panel := lipgloss.JoinVertical(lipgloss.Left, title, content.String())

	return panelStyle.Width(p.width - 2).Height(p.height - 2).Render(panel)
}

func (p *OrderInputPanel) renderField(label string, field OrderInputField, inputView string) string {
	labelStyle := styles.LabelStyle
	if p.currentField == field && p.focused {
		labelStyle = labelStyle.Foreground(styles.PrimaryColor)
	}
	labelStr := labelStyle.Render(fmt.Sprintf("%-8s", label))
	return labelStr + inputView
}

func (p *OrderInputPanel) renderTickerField() string {
	var result strings.Builder

	// Render input
	inputStyle := styles.InputStyle
	if p.currentField == FieldTicker && p.focused {
		inputStyle = styles.FocusedInputStyle
		p.tickerInput.Focus()
	} else {
		p.tickerInput.Blur()
	}

	result.WriteString(inputStyle.Render(p.tickerInput.View()))

	// Render dropdown if showing
	if p.showDropdown && len(p.dropdownFiltered) > 0 {
		result.WriteString("\n")
		maxShow := 5
		if len(p.dropdownFiltered) < maxShow {
			maxShow = len(p.dropdownFiltered)
		}

		for i := 0; i < maxShow; i++ {
			item := p.dropdownFiltered[i]
			style := styles.DropdownItemStyle
			if i == p.dropdownIndex {
				style = styles.DropdownSelectedStyle
			}

			// Highlight matching characters
			highlighted := p.highlightMatch(item, p.tickerInput.Value())
			result.WriteString("         " + style.Render(highlighted))
			if i < maxShow-1 {
				result.WriteString("\n")
			}
		}
	}

	return result.String()
}

func (p *OrderInputPanel) renderSideField() string {
	var items []string
	for i, opt := range p.sideOptions {
		style := styles.DropdownItemStyle
		if i == p.sideIndex {
			if p.currentField == FieldSide && p.focused {
				style = styles.DropdownSelectedStyle
			} else {
				style = styles.DropdownItemStyle.Bold(true)
			}
		}

		// Color code buy/sell
		if opt == "BUY" && i == p.sideIndex {
			style = style.Foreground(styles.BuyColor)
		} else if opt == "SELL" && i == p.sideIndex {
			style = style.Foreground(styles.SellColor)
		}

		items = append(items, style.Render(opt))
	}
	return strings.Join(items, " | ")
}

func (p *OrderInputPanel) renderTypeField() string {
	var items []string
	for i, opt := range p.typeOptions {
		style := styles.DropdownItemStyle
		if i == p.typeIndex {
			if p.currentField == FieldType && p.focused {
				style = styles.DropdownSelectedStyle
			} else {
				style = styles.DropdownItemStyle.Bold(true)
			}
		}
		items = append(items, style.Render(opt))
	}
	return strings.Join(items, " | ")
}

func (p *OrderInputPanel) renderOrderSummary() string {
	var parts []string

	// Ticker
	ticker := p.tickerInput.Value()
	if p.selectedTicker != nil {
		ticker = p.selectedTicker.Name
	}
	if ticker == "" {
		ticker = "---"
	}
	parts = append(parts, ticker)

	// Side
	side := p.sideOptions[p.sideIndex]
	sideStyle := styles.BuyStyle
	if side == "SELL" {
		sideStyle = styles.SellStyle
	}
	parts = append(parts, sideStyle.Render(side))

	// Type
	parts = append(parts, p.typeOptions[p.typeIndex])

	// Price (for limit orders)
	if p.typeIndex == 0 {
		price := p.priceInput.Value()
		if price == "" {
			price = "0"
		}
		parts = append(parts, "@"+price)
	}

	// Quantity
	qty := p.quantityInput.Value()
	if qty == "" {
		qty = "0"
	}
	parts = append(parts, "x"+qty)

	return styles.HeaderStyle.Render("Order: ") + strings.Join(parts, " ")
}

func (p *OrderInputPanel) filterDropdown(query string) {
	query = strings.ToUpper(query)
	p.dropdownFiltered = nil
	p.dropdownIndex = 0

	for _, item := range p.dropdownItems {
		if strings.Contains(strings.ToUpper(item), query) {
			p.dropdownFiltered = append(p.dropdownFiltered, item)
		}
	}
}

func (p *OrderInputPanel) highlightMatch(item, query string) string {
	if query == "" {
		return item
	}

	upper := strings.ToUpper(item)
	queryUpper := strings.ToUpper(query)
	idx := strings.Index(upper, queryUpper)
	if idx == -1 {
		return item
	}

	before := item[:idx]
	match := item[idx : idx+len(query)]
	after := item[idx+len(query):]

	return before + styles.DropdownMatchStyle.Render(match) + after
}

func (p *OrderInputPanel) selectDropdownItem() {
	if p.dropdownIndex < len(p.dropdownFiltered) {
		selected := p.dropdownFiltered[p.dropdownIndex]
		p.tickerInput.SetValue(selected)

		// Find and set the actual ticker
		for i, t := range p.tickers {
			if t.Name == selected {
				p.selectedTicker = &p.tickers[i]
				break
			}
		}
	}
}

func (p *OrderInputPanel) nextField() {
	p.showDropdown = false
	switch p.currentField {
	case FieldTicker:
		p.selectDropdownItem()
		p.currentField = FieldSide
		p.tickerInput.Blur()
	case FieldSide:
		p.currentField = FieldType
	case FieldType:
		if p.typeIndex == 0 { // LIMIT
			p.currentField = FieldPrice
			p.priceInput.Focus()
		} else {
			p.currentField = FieldQuantity
			p.quantityInput.Focus()
		}
	case FieldPrice:
		p.currentField = FieldQuantity
		p.priceInput.Blur()
		p.quantityInput.Focus()
	case FieldQuantity:
		p.currentField = FieldSubmit
		p.quantityInput.Blur()
	case FieldSubmit:
		p.currentField = FieldTicker
		p.tickerInput.Focus()
	}
}

func (p *OrderInputPanel) prevField() {
	p.showDropdown = false
	switch p.currentField {
	case FieldTicker:
		p.currentField = FieldSubmit
		p.tickerInput.Blur()
	case FieldSide:
		p.currentField = FieldTicker
		p.tickerInput.Focus()
	case FieldType:
		p.currentField = FieldSide
	case FieldPrice:
		p.currentField = FieldType
		p.priceInput.Blur()
	case FieldQuantity:
		if p.typeIndex == 0 { // LIMIT
			p.currentField = FieldPrice
			p.priceInput.Focus()
		} else {
			p.currentField = FieldType
		}
		p.quantityInput.Blur()
	case FieldSubmit:
		p.currentField = FieldQuantity
		p.quantityInput.Focus()
	}
}

func (p *OrderInputPanel) submitOrder() tea.Cmd {
	// Validate inputs
	if p.selectedTicker == nil {
		return nil
	}

	qty, err := strconv.ParseInt(p.quantityInput.Value(), 10, 64)
	if err != nil || qty <= 0 {
		return nil
	}

	side := core.SideBuy
	if p.sideIndex == 1 {
		side = core.SideSell
	}

	orderKind := core.OrderKindLimit
	if p.typeIndex == 1 {
		orderKind = core.OrderKindMarket
	}

	var price int64
	if orderKind == core.OrderKindLimit {
		price, err = strconv.ParseInt(p.priceInput.Value(), 10, 64)
		if err != nil || price <= 0 {
			return nil
		}
	}

	// Create and return submit command
	return func() tea.Msg {
		return OrderSubmitMsg{
			Ticker:    *p.selectedTicker,
			Side:      side,
			OrderKind: orderKind,
			Price:     core.PriceTicks(price),
			Quantity:  core.Size(qty),
		}
	}
}

// SetFocus sets the focus state of the panel.
func (p *OrderInputPanel) SetFocus(focused bool) {
	p.focused = focused
	if focused {
		switch p.currentField {
		case FieldTicker:
			p.tickerInput.Focus()
		case FieldPrice:
			p.priceInput.Focus()
		case FieldQuantity:
			p.quantityInput.Focus()
		}
	} else {
		p.tickerInput.Blur()
		p.priceInput.Blur()
		p.quantityInput.Blur()
	}
}

// SetSize sets the panel dimensions.
func (p *OrderInputPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetTicker pre-fills the ticker field.
func (p *OrderInputPanel) SetTicker(ticker market.Ticker) {
	p.tickerInput.SetValue(ticker.Name)
	p.selectedTicker = &ticker
}

// Reset clears the input fields.
func (p *OrderInputPanel) Reset() {
	p.tickerInput.SetValue("")
	p.priceInput.SetValue("")
	p.quantityInput.SetValue("")
	p.selectedTicker = nil
	p.currentField = FieldTicker
	p.sideIndex = 0
	p.typeIndex = 0
	p.showDropdown = false
}

// OrderSubmitMsg is sent when an order is submitted.
type OrderSubmitMsg struct {
	Ticker    market.Ticker
	Side      core.Side
	OrderKind core.OrderKind
	Price     core.PriceTicks
	Quantity  core.Size
}
