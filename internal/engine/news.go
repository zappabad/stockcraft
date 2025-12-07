package engine

import (
	"fmt"
	"math/rand"
)

type SymbolImpact struct {
	Symbol string
	Impact float64 // e.g., percentage change in price
}

type News struct {
	ID            string
	Headline      string
	SymbolImpacts []SymbolImpact
}

type NewsEngine struct {
	newsItems []News
}

func NewNewsEngine() *NewsEngine {
	return &NewsEngine{
		newsItems: []News{},
	}
}

func (ne *NewsEngine) GenerateNews(tick int) *News {
	// Simple example: generate news every 10 ticks
	if tick%10 == 0 {
		impact_sign := rand.Intn(3) - 1 // -1, 0, or +1
		news := News{
			ID:       fmt.Sprintf("news-%d", tick),
			Headline: fmt.Sprintf("Breaking News at tick %d!", tick),
			SymbolImpacts: []SymbolImpact{
				{Symbol: "FOO", Impact: 0.05 * float64(impact_sign)},
				{Symbol: "BAR", Impact: -0.05 * float64(impact_sign)},
			},
		}
		ne.newsItems = append(ne.newsItems, news)
		return &news
	}
	return nil
}

func (ne *NewsEngine) GetNews() *News {
	if len(ne.newsItems) == 0 {
		return nil
	}
	return &ne.newsItems[len(ne.newsItems)-1]
}
