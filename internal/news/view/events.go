package view

import "github.com/zappabad/stockcraft/internal/news"

// NewsEvent wraps a news item for the event channel.
type NewsEvent struct {
	Item news.NewsItem
}
