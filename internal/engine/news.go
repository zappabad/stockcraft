package engine

import (
	"fmt"
	"math/rand"
)

type (
	TopicID  string
	Ticker   string
	Headline string
	Content  string

	// TopicNode represents a node in the topic tree.
	// Leaf nodes typically refer to specific tickers.
	TopicNode struct {
		ID       TopicID
		Name     string
		Children []*TopicNode
		Tickers  []Ticker
		// TODO: add parent pointer or path if needed for navigation.
	}

	// TopicSubscription represents how much an agent cares about a topic.
	TopicSubscription struct {
		Topic  TopicID
		Weight float64 // relative importance for this agent
		// TODO: add fields like "delay", "credibility", etc.
	}

	NewsDetails struct {
		Headline Headline
		Content  Content
	}

	// News is a single news item affecting a topic and some tickers.
	News struct {
		ID        int64
		Topic     TopicID
		Affected  []Ticker
		Timestamp int64   // simulation time
		Horizon   int64   // how far into the future this refers
		Sentiment float64 // e.g. [-1, 1]
		Magnitude float64 // absolute strength of the news
		Details   NewsDetails

		// TODO: add fields like source, confidence, tags, etc.
	}

	NewsEngine struct {
		newsItems []News
	}
)

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
			ID:        int64(len(ne.newsItems) + 1),
			Topic:     TopicID(fmt.Sprintf("Topic-%d", rand.Intn(5))),
			Affected:  []Ticker{Ticker(fmt.Sprintf("SYM%d", rand.Intn(10)))},
			Timestamp: int64(tick),
			Horizon:   int64(tick + rand.Intn(5) + 1),
			Sentiment: float64(impact_sign),
			Magnitude: float64(rand.Intn(100)) / 10.0,
			Details: NewsDetails{
				Headline: Headline("Breaking News!"),
				Content:  Content("Something significant has happened."),
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
