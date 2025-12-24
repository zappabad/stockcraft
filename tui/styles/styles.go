package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Primary colors
	PrimaryColor   = lipgloss.Color("#7C3AED") // Purple
	SecondaryColor = lipgloss.Color("#10B981") // Green
	AccentColor    = lipgloss.Color("#F59E0B") // Amber

	// Status colors
	BuyColor     = lipgloss.Color("#10B981") // Green
	SellColor    = lipgloss.Color("#EF4444") // Red
	NeutralColor = lipgloss.Color("#6B7280") // Gray

	// Background colors
	BackgroundColor      = lipgloss.Color("#1F2937")
	PanelBackgroundColor = lipgloss.Color("#111827")
	BorderColor          = lipgloss.Color("#374151")
	FocusBorderColor     = lipgloss.Color("#7C3AED")

	// Text colors
	TextColor          = lipgloss.Color("#F9FAFB")
	TextSecondaryColor = lipgloss.Color("#9CA3AF")
	TextMutedColor     = lipgloss.Color("#6B7280")
)

// Panel styles
var (
	// Base panel style
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	// Focused panel style
	FocusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(FocusBorderColor).
				Padding(0, 1)

	// Panel title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			Padding(0, 1)

	// Header row style
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextSecondaryColor)

	// Row styles
	RowStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	SelectedRowStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Background(lipgloss.Color("#374151"))
)

// Text styles
var (
	// Buy/Sell text
	BuyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(BuyColor)

	SellStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(SellColor)

	// Price styles
	PriceStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	PriceUpStyle = lipgloss.NewStyle().
			Foreground(BuyColor)

	PriceDownStyle = lipgloss.NewStyle().
			Foreground(SellColor)

	// Size style
	SizeStyle = lipgloss.NewStyle().
			Foreground(TextSecondaryColor)

	// Timestamp style
	TimeStyle = lipgloss.NewStyle().
			Foreground(TextMutedColor)

	// News severity styles
	NewsNormalStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	NewsImportantStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(AccentColor)
)

// Input styles
var (
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(FocusBorderColor).
				Padding(0, 1)

	LabelStyle = lipgloss.NewStyle().
			Foreground(TextSecondaryColor)

	PlaceholderStyle = lipgloss.NewStyle().
				Foreground(TextMutedColor)

	DropdownStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(BorderColor).
			Background(PanelBackgroundColor)

	DropdownItemStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Padding(0, 1)

	DropdownSelectedStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Background(lipgloss.Color("#374151")).
				Padding(0, 1)

	DropdownMatchStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Bold(true)
)

// Chart styles (for candlestick)
var (
	CandleUpStyle = lipgloss.NewStyle().
			Foreground(BuyColor)

	CandleDownStyle = lipgloss.NewStyle().
			Foreground(SellColor)

	ChartAxisStyle = lipgloss.NewStyle().
			Foreground(TextMutedColor)

	ChartLabelStyle = lipgloss.NewStyle().
			Foreground(TextSecondaryColor)
)

// Status bar styles
var (
	StatusBarStyle = lipgloss.NewStyle().
			Background(BackgroundColor).
			Foreground(TextSecondaryColor).
			Padding(0, 1)

	StatusBarKeyStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Bold(true)

	StatusBarDescStyle = lipgloss.NewStyle().
				Foreground(TextSecondaryColor)
)

// Helper function to render a title bar for a panel
func RenderTitle(title string, focused bool) string {
	style := TitleStyle
	if focused {
		style = style.Foreground(FocusBorderColor)
	}
	return style.Render(title)
}

// Helper to format price with ticker decimals
func FormatPrice(price int64, decimals int8) string {
	if decimals <= 0 {
		return fmt.Sprintf("%d", price)
	}
	// Simple decimal formatting
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
