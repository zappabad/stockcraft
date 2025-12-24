package panels

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zappabad/stockcraft/internal/market"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
	"github.com/zappabad/stockcraft/tui/styles"
)

// Candle represents a single candlestick.
type Candle struct {
	Open   core.PriceTicks
	High   core.PriceTicks
	Low    core.PriceTicks
	Close  core.PriceTicks
	Volume core.Size
	Time   int64
}

// CandlestickPanel displays a candlestick chart.
type CandlestickPanel struct {
	ticker  market.Ticker
	candles []Candle

	// Current candle being built
	currentCandle *Candle
	candleStart   int64
	candlePeriod  int64 // in nanoseconds (e.g., 1 second = 1e9)

	focused bool
	width   int
	height  int

	// Chart settings
	maxCandles int
}

// NewCandlestickPanel creates a new candlestick chart panel.
func NewCandlestickPanel() *CandlestickPanel {
	return &CandlestickPanel{
		candlePeriod: 5e9, // 5 second candles
		maxCandles:   50,
	}
}

// Init initializes the panel.
func (p *CandlestickPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the panel.
func (p *CandlestickPanel) Update(msg tea.Msg) (*CandlestickPanel, tea.Cmd) {
	return p, nil
}

// View renders the panel.
func (p *CandlestickPanel) View() string {
	tickerName := "No ticker"
	if p.ticker.Name != "" {
		tickerName = p.ticker.Name
	}

	var content strings.Builder

	// Calculate chart dimensions
	chartWidth := p.width - 12 // Leave room for price axis
	chartHeight := p.height - 6
	if chartHeight < 5 {
		chartHeight = 5
	}

	// Get all candles including current one being built
	allCandles := p.getAllCandles()

	if len(allCandles) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(styles.TextMutedColor).Render("No trading data yet..."))
	} else {
		content.WriteString(p.renderChart(chartWidth, chartHeight, allCandles))
	}

	// Apply panel styling
	panelStyle := styles.PanelStyle
	if p.focused {
		panelStyle = styles.FocusedPanelStyle
	}

	title := styles.RenderTitle(fmt.Sprintf("📉 Chart - %s", tickerName), p.focused)
	panel := lipgloss.JoinVertical(lipgloss.Left, title, content.String())

	return panelStyle.Width(p.width - 2).Height(p.height - 2).Render(panel)
}

// getAllCandles returns all candles including the current one being built.
func (p *CandlestickPanel) getAllCandles() []Candle {
	if p.currentCandle == nil {
		return p.candles
	}
	// Include current candle in the view
	return append(p.candles, *p.currentCandle)
}

func (p *CandlestickPanel) renderChart(width, height int, candles []Candle) string {
	if len(candles) == 0 {
		return ""
	}

	// Reserve space: 9 chars for price axis, 1 for separator
	chartWidth := width - 10
	if chartWidth < 10 {
		chartWidth = 10
	}

	// Each candle needs 3 chars: space, candle, space
	candleWidth := 3
	candlesToShow := chartWidth / candleWidth
	if candlesToShow < 1 {
		candlesToShow = 1
	}
	if candlesToShow > len(candles) {
		candlesToShow = len(candles)
	}

	// Get the most recent candles
	displayCandles := candles
	if len(candles) > candlesToShow {
		displayCandles = candles[len(candles)-candlesToShow:]
	}

	// Find price range
	minPrice := displayCandles[0].Low
	maxPrice := displayCandles[0].High
	for _, c := range displayCandles {
		if c.Low < minPrice {
			minPrice = c.Low
		}
		if c.High > maxPrice {
			maxPrice = c.High
		}
	}

	// Add padding to price range (10%)
	priceRange := maxPrice - minPrice
	if priceRange == 0 {
		priceRange = 100 // Minimum range
	}
	padding := core.PriceTicks(float64(priceRange) * 0.1)
	if padding < 1 {
		padding = 1
	}
	minPrice -= padding
	maxPrice += padding

	// Reserve 2 rows for time axis
	chartHeight := height - 3
	if chartHeight < 5 {
		chartHeight = 5
	}

	var result strings.Builder

	// Render chart rows (top to bottom = high to low price)
	for row := 0; row < chartHeight; row++ {
		// Price label
		price := p.yToPrice(row, minPrice, maxPrice, chartHeight)
		priceLabel := formatPrice(int64(price), p.ticker.Decimals)
		result.WriteString(styles.ChartAxisStyle.Render(fmt.Sprintf("%8s │", priceLabel)))

		// Render each candle column
		for _, candle := range displayCandles {
			char := p.getCandleChar(candle, row, minPrice, maxPrice, chartHeight)

			// Apply color based on bullish/bearish
			var style lipgloss.Style
			if candle.Close >= candle.Open {
				style = styles.CandleUpStyle
			} else {
				style = styles.CandleDownStyle
			}

			result.WriteString(style.Render(string(char)))
			result.WriteString(" ") // Space between candles
		}
		result.WriteString("\n")
	}

	// Bottom border
	result.WriteString(styles.ChartAxisStyle.Render("─────────┴"))
	for range displayCandles {
		result.WriteString(styles.ChartAxisStyle.Render("──"))
	}
	result.WriteString("\n")

	// Time axis - show relative time labels
	result.WriteString(styles.ChartAxisStyle.Render("          "))
	for i, candle := range displayCandles {
		// Show time every few candles or for first/last
		if i == 0 || i == len(displayCandles)-1 || i%5 == 0 {
			t := time.Unix(0, candle.Time)
			timeStr := t.Format("04")
			result.WriteString(styles.ChartLabelStyle.Render(timeStr))
		} else {
			result.WriteString("  ")
		}
	}

	return result.String()
}

// getCandleChar returns the character to draw for a candle at a given row
func (p *CandlestickPanel) getCandleChar(candle Candle, row int, minPrice, maxPrice core.PriceTicks, height int) rune {
	// Convert row to price level
	rowPrice := p.yToPrice(row, minPrice, maxPrice, height)

	// Get candle price positions
	highPrice := candle.High
	lowPrice := candle.Low

	bodyTop := candle.Open
	bodyBottom := candle.Close
	if candle.Close > candle.Open {
		bodyTop = candle.Close
		bodyBottom = candle.Open
	}

	// Check if this row intersects with the candle
	// We need some tolerance since we're mapping continuous prices to discrete rows
	tolerance := (maxPrice - minPrice) / core.PriceTicks(height*2)
	if tolerance < 1 {
		tolerance = 1
	}

	// Check body first (body overwrites wick)
	if rowPrice <= bodyTop+tolerance && rowPrice >= bodyBottom-tolerance {
		if candle.Close >= candle.Open {
			return '┃' // Bullish body (thick)
		}
		return '┃' // Bearish body (same char, different color)
	}

	// Check upper wick (above body)
	if rowPrice <= highPrice+tolerance && rowPrice > bodyTop {
		return '│' // Thin wick
	}

	// Check lower wick (below body)
	if rowPrice >= lowPrice-tolerance && rowPrice < bodyBottom {
		return '│' // Thin wick
	}

	return ' ' // Empty space
}

func (p *CandlestickPanel) priceToY(price, minPrice, maxPrice core.PriceTicks, height int) int {
	if maxPrice == minPrice {
		return height / 2
	}
	ratio := float64(maxPrice-price) / float64(maxPrice-minPrice)
	y := int(ratio * float64(height-1))
	if y < 0 {
		y = 0
	}
	if y >= height {
		y = height - 1
	}
	return y
}

func (p *CandlestickPanel) yToPrice(y int, minPrice, maxPrice core.PriceTicks, height int) core.PriceTicks {
	if height <= 1 {
		return minPrice
	}
	ratio := float64(y) / float64(height-1)
	return maxPrice - core.PriceTicks(ratio*float64(maxPrice-minPrice))
}

// SetFocus sets the focus state of the panel.
func (p *CandlestickPanel) SetFocus(focused bool) {
	p.focused = focused
}

// SetSize sets the panel dimensions.
func (p *CandlestickPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetTicker sets the ticker to chart.
func (p *CandlestickPanel) SetTicker(ticker market.Ticker) {
	p.ticker = ticker
	p.candles = nil
	p.currentCandle = nil
}

// AddTrade processes a trade and updates the candlestick data.
func (p *CandlestickPanel) AddTrade(trade core.TradeEvent) {
	// Check if we need to start a new candle
	candleStart := (trade.Time / p.candlePeriod) * p.candlePeriod

	if p.currentCandle == nil || candleStart != p.candleStart {
		// Finalize current candle if exists
		if p.currentCandle != nil {
			p.candles = append(p.candles, *p.currentCandle)
			// Keep only maxCandles
			if len(p.candles) > p.maxCandles {
				p.candles = p.candles[len(p.candles)-p.maxCandles:]
			}
		}

		// Start new candle
		p.currentCandle = &Candle{
			Open:   trade.Price,
			High:   trade.Price,
			Low:    trade.Price,
			Close:  trade.Price,
			Volume: trade.Size,
			Time:   candleStart,
		}
		p.candleStart = candleStart
	} else {
		// Update current candle
		if trade.Price > p.currentCandle.High {
			p.currentCandle.High = trade.Price
		}
		if trade.Price < p.currentCandle.Low {
			p.currentCandle.Low = trade.Price
		}
		p.currentCandle.Close = trade.Price
		p.currentCandle.Volume += trade.Size
	}
}

// SetCandles sets the candle data directly.
func (p *CandlestickPanel) SetCandles(candles []Candle) {
	p.candles = candles
}

// GenerateSampleCandles generates sample candle data for testing.
func (p *CandlestickPanel) GenerateSampleCandles(basePrice int64, count int) {
	p.candles = nil
	price := float64(basePrice)

	for i := 0; i < count; i++ {
		// Random walk
		change := (float64(i%7) - 3) / 100 * price

		open := core.PriceTicks(price)
		price += change
		closePrice := core.PriceTicks(price)

		high := open
		if closePrice > high {
			high = closePrice
		}
		high += core.PriceTicks(math.Abs(change) * 0.5)

		low := open
		if closePrice < low {
			low = closePrice
		}
		low -= core.PriceTicks(math.Abs(change) * 0.5)

		p.candles = append(p.candles, Candle{
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: core.Size(100 + i%50),
			Time:   int64(i) * p.candlePeriod,
		})
	}
}

// Ticker returns the current ticker.
func (p *CandlestickPanel) Ticker() market.Ticker {
	return p.ticker
}
