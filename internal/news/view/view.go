package view

import (
	"sync"

	"github.com/zappabad/stockcraft/internal/news"
)

// NewsView maintains a bounded ring buffer of news items.
type NewsView struct {
	mu    sync.RWMutex
	buf   []news.NewsItem
	size  int
	start int
	count int
}

// NewNewsView creates a new NewsView with the given capacity.
func NewNewsView(capacity int) *NewsView {
	if capacity <= 0 {
		capacity = 100
	}
	return &NewsView{
		buf:  make([]news.NewsItem, capacity),
		size: capacity,
	}
}

// Apply adds a news item to the view.
func (v *NewsView) Apply(ev NewsEvent) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.count < v.size {
		v.buf[(v.start+v.count)%v.size] = ev.Item
		v.count++
		return
	}
	// overwrite oldest
	v.buf[v.start] = ev.Item
	v.start = (v.start + 1) % v.size
}

// Latest returns the last n news items in chronological order (oldest first).
// Returns a copy (not internal references).
func (v *NewsView) Latest(n int) []news.NewsItem {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if n <= 0 || v.count == 0 {
		return nil
	}
	if n > v.count {
		n = v.count
	}

	out := make([]news.NewsItem, n)
	// take last n in chronological order
	first := (v.start + (v.count - n)) % v.size
	for i := 0; i < n; i++ {
		out[i] = v.buf[(first+i)%v.size]
	}
	return out
}

// Count returns the number of news items in the view.
func (v *NewsView) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.count
}
