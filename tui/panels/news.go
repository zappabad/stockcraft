package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zappabad/stockcraft/internal/news"
	"github.com/zappabad/stockcraft/tui/styles"
)

// NewsPanel displays news items.
type NewsPanel struct {
	news          []news.NewsItem
	selectedIndex int
	scrollOffset  int
	focused       bool
	width         int
	height        int
	maxItems      int
}

// NewNewsPanel creates a new news panel.
func NewNewsPanel() *NewsPanel {
	return &NewsPanel{
		maxItems: 50,
	}
}

// Init initializes the panel.
func (p *NewsPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the panel.
func (p *NewsPanel) Update(msg tea.Msg) (*NewsPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if p.selectedIndex > 0 {
				p.selectedIndex--
				// Adjust scroll to keep selection in view
				if p.selectedIndex < p.scrollOffset {
					p.scrollOffset = p.selectedIndex
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if p.selectedIndex < len(p.news)-1 {
				p.selectedIndex++
				// Adjust scroll to keep selection in view
				visibleItems := p.height - 4
				if p.selectedIndex >= p.scrollOffset+visibleItems {
					p.scrollOffset = p.selectedIndex - visibleItems + 1
				}
			}
		}
	}
	return p, nil
}

// View renders the panel.
func (p *NewsPanel) View() string {
	var content strings.Builder

	if len(p.news) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(styles.TextMutedColor).Render("No news available"))
	} else {
		// Calculate visible items
		visibleItems := p.height - 4
		if visibleItems < 1 {
			visibleItems = 1
		}

		start := p.scrollOffset
		end := start + visibleItems
		if end > len(p.news) {
			end = len(p.news)
		}

		for i := start; i < end; i++ {
			item := p.news[i]

			// Format time
			t := time.Unix(0, item.Time)
			timeStr := t.Format("15:04:05")

			// Build headline line
			headline := item.Headline
			if len(headline) > p.width-15 {
				headline = headline[:p.width-18] + "..."
			}

			// Choose style based on severity
			var headlineStyle lipgloss.Style
			if item.Severity > 0 {
				headlineStyle = styles.NewsImportantStyle
			} else {
				headlineStyle = styles.NewsNormalStyle
			}

			// Select styling
			timeStyled := styles.TimeStyle.Render(timeStr)
			headlineStyled := headlineStyle.Render(headline)

			line := fmt.Sprintf("%s %s", timeStyled, headlineStyled)

			if i == p.selectedIndex && p.focused {
				line = styles.SelectedRowStyle.Render(line)
			}

			content.WriteString(line)
			if i < end-1 {
				content.WriteString("\n")
			}
		}

		// Scroll indicator
		if len(p.news) > visibleItems {
			scrollInfo := fmt.Sprintf(" (%d/%d)", p.selectedIndex+1, len(p.news))
			content.WriteString("\n")
			content.WriteString(lipgloss.NewStyle().Foreground(styles.TextMutedColor).Render(scrollInfo))
		}
	}

	// Apply panel styling
	panelStyle := styles.PanelStyle
	if p.focused {
		panelStyle = styles.FocusedPanelStyle
	}

	title := styles.RenderTitle("📰 News", p.focused)
	panel := lipgloss.JoinVertical(lipgloss.Left, title, content.String())

	return panelStyle.Width(p.width - 2).Height(p.height - 2).Render(panel)
}

// SetFocus sets the focus state of the panel.
func (p *NewsPanel) SetFocus(focused bool) {
	p.focused = focused
}

// SetSize sets the panel dimensions.
func (p *NewsPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetNews sets the news items.
func (p *NewsPanel) SetNews(items []news.NewsItem) {
	p.news = items
	// Reset selection if out of bounds
	if p.selectedIndex >= len(p.news) {
		p.selectedIndex = len(p.news) - 1
		if p.selectedIndex < 0 {
			p.selectedIndex = 0
		}
	}
}

// AddNews adds a news item to the panel.
func (p *NewsPanel) AddNews(item news.NewsItem) {
	p.news = append(p.news, item)
	// Keep only maxItems
	if len(p.news) > p.maxItems {
		p.news = p.news[len(p.news)-p.maxItems:]
	}
}

// SelectedNews returns the currently selected news item.
func (p *NewsPanel) SelectedNews() *news.NewsItem {
	if p.selectedIndex >= 0 && p.selectedIndex < len(p.news) {
		return &p.news[p.selectedIndex]
	}
	return nil
}

// NewsUpdateMsg is sent when news is received.
type NewsUpdateMsg struct {
	Item news.NewsItem
}
