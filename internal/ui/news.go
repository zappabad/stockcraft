package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zappabad/stockcraft/internal/engine"
)

// NewsWidget displays scrolling news feed
type NewsWidget struct {
	BaseWidget
	newsItems []engine.News
	lastTick  int
	maxItems  int
}

func NewNewsWidget() *NewsWidget {
	return &NewsWidget{
		BaseWidget: NewBaseWidget(),
		newsItems:  make([]engine.News, 0),
		lastTick:   0,
		maxItems:   10, // Keep last 10 news items
	}
}

func (w *NewsWidget) Update(event UIEvent) bool {
	if newsEvent, ok := event.(NewsUpdateEvent); ok {
		if newsEvent.News != nil {
			// Add new news item
			w.newsItems = append(w.newsItems, *newsEvent.News)

			// Keep only last maxItems
			if len(w.newsItems) > w.maxItems {
				w.newsItems = w.newsItems[len(w.newsItems)-w.maxItems:]
			}

			w.lastTick = newsEvent.Tick
			return true
		}
	}
	return false
}

func (w *NewsWidget) Render(width, height int) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	if w.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("32"))
	}

	// Build header
	header := "News Feed"

	// Build news display
	var lines []string
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", len(header)))

	if len(w.newsItems) == 0 {
		lines = append(lines, "No news available")
	} else {
		// Calculate available lines for content (subtract header + separator + border)
		availableLines := height - 4 // 2 for border, 2 for header+separator
		if availableLines < 1 {
			availableLines = 1
		}

		currentLines := 0
		// Show news items in reverse order (newest first)
		for i := len(w.newsItems) - 1; i >= 0 && currentLines < availableLines; i-- {
			news := w.newsItems[i]

			// Check if we have space for this news item
			itemLines := 1 // headline
			if len(news.Magnitude) > 0 {
				itemLines++ // impact line
			}
			if i > 0 {
				itemLines++ // empty line separator
			}

			if currentLines+itemLines > availableLines {
				break
			}

			// Truncate headline if too long
			headline := news.Headline
			maxHeadlineWidth := width - 6 // Account for border, padding, and bullet
			if maxHeadlineWidth < 10 {
				maxHeadlineWidth = 10
			}
			if len(headline) > maxHeadlineWidth {
				headline = headline[:maxHeadlineWidth-3] + "..."
			}

			lines = append(lines, fmt.Sprintf("• %s", headline))
			currentLines++

			// Show symbol impacts
			if len(news.SymbolImpacts) > 0 && currentLines < availableLines {
				var impacts []string
				for _, impact := range news.SymbolImpacts {
					if impact.Impact > 0 {
						impacts = append(impacts, fmt.Sprintf("%s +%.1f%%", impact.Symbol, impact.Impact*100))
					} else if impact.Impact < 0 {
						impacts = append(impacts, fmt.Sprintf("%s %.1f%%", impact.Symbol, impact.Impact*100))
					}
				}
				if len(impacts) > 0 {
					impactStr := strings.Join(impacts, ", ")
					// Truncate impact string if too long
					maxImpactWidth := width - 12 // Account for "  Impact: " prefix
					if len(impactStr) > maxImpactWidth {
						impactStr = impactStr[:maxImpactWidth-3] + "..."
					}
					lines = append(lines, fmt.Sprintf("  Impact: %s", impactStr))
					currentLines++
				}
			}

			// Add empty line between news items (if not the last and we have space)
			if i > 0 && currentLines < availableLines {
				lines = append(lines, "")
				currentLines++
			}
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
