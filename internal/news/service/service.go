package service

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/zappabad/stockcraft/internal/news"
	newsview "github.com/zappabad/stockcraft/internal/news/view"
)

// NewsService manages news publishing and viewing.
type NewsService struct {
	cfg  Config
	view *newsview.NewsView

	idGen atomic.Int64

	internalEvents chan newsview.NewsEvent
	externalEvents chan newsview.NewsEvent
	droppedEvents  atomic.Int64

	closed    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewNewsService creates a new NewsService.
func NewNewsService(cfg Config) *NewsService {
	if cfg.TapeSize <= 0 {
		cfg.TapeSize = DefaultConfig().TapeSize
	}
	if cfg.EventBuffer <= 0 {
		cfg.EventBuffer = DefaultConfig().EventBuffer
	}
	if cfg.ExternalEventBuffer <= 0 {
		cfg.ExternalEventBuffer = DefaultConfig().ExternalEventBuffer
	}

	s := &NewsService{
		cfg:            cfg,
		view:           newsview.NewNewsView(cfg.TapeSize),
		internalEvents: make(chan newsview.NewsEvent, cfg.EventBuffer),
		externalEvents: make(chan newsview.NewsEvent, cfg.ExternalEventBuffer),
		closed:         make(chan struct{}),
	}

	// Initialize ID generator
	s.idGen.Store(time.Now().UnixNano())

	// Start event dispatcher
	s.wg.Add(1)
	go s.runEventDispatcher()

	return s
}

func (s *NewsService) nextID() news.NewsID {
	return news.NewsID(s.idGen.Add(1))
}

func (s *NewsService) runEventDispatcher() {
	defer s.wg.Done()
	defer close(s.externalEvents)

	for {
		select {
		case <-s.closed:
			return
		case ev := <-s.internalEvents:
			// Always update view (authoritative)
			s.view.Apply(ev)

			// Attempt to send to external channel
			if s.cfg.DropExternalEvents {
				select {
				case s.externalEvents <- ev:
				default:
					s.droppedEvents.Add(1)
				}
			} else {
				select {
				case s.externalEvents <- ev:
				case <-s.closed:
					return
				}
			}
		}
	}
}

// Publish publishes a news item. Sets ID and Time if missing.
func (s *NewsService) Publish(item news.NewsItem) {
	if item.ID == 0 {
		item.ID = s.nextID()
	}
	if item.Time == 0 {
		item.Time = time.Now().UnixNano()
	}

	ev := newsview.NewsEvent{Item: item}

	select {
	case s.internalEvents <- ev:
	case <-s.closed:
	}
}

// Latest returns the last n news items (from view).
func (s *NewsService) Latest(n int) []news.NewsItem {
	return s.view.Latest(n)
}

// Events returns the external events channel for subscribers.
func (s *NewsService) Events() <-chan newsview.NewsEvent {
	return s.externalEvents
}

// DroppedEvents returns the count of dropped external events.
func (s *NewsService) DroppedEvents() int64 {
	return s.droppedEvents.Load()
}

// Close shuts down the news service.
func (s *NewsService) Close() {
	s.closeOnce.Do(func() {
		close(s.closed)
	})
	s.wg.Wait()
}
