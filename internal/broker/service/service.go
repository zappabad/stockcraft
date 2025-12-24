package service

import (
	"sync"

	"github.com/zappabad/stockcraft/internal/broker"
	brokerview "github.com/zappabad/stockcraft/internal/broker/view"
	newsview "github.com/zappabad/stockcraft/internal/news/view"
	"github.com/zappabad/stockcraft/internal/trader"
)

// BrokerService manages broker state and handles trader/news events.
type BrokerService struct {
	cfg  Config
	view *brokerview.BrokerView

	closed    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewBrokerService creates a new BrokerService.
func NewBrokerService(cfg Config) *BrokerService {
	if cfg.RequestCapacity <= 0 {
		cfg.RequestCapacity = DefaultConfig().RequestCapacity
	}

	return &BrokerService{
		cfg:    cfg,
		view:   brokerview.NewBrokerView(cfg.RequestCapacity),
		closed: make(chan struct{}),
	}
}

// AttachTraderEvents starts listening to trader events in a goroutine.
func (s *BrokerService) AttachTraderEvents(events <-chan trader.TraderEvent) {
	s.wg.Add(1)
	go s.runTraderEventListener(events)
}

func (s *BrokerService) runTraderEventListener(events <-chan trader.TraderEvent) {
	defer s.wg.Done()

	for {
		select {
		case <-s.closed:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			s.handleTraderEvent(ev)
		}
	}
}

func (s *BrokerService) handleTraderEvent(ev trader.TraderEvent) {
	// Only handle approval requests for now
	if ev.Type == trader.TraderEventRequestedApproval {
		req := broker.Request{
			TraderID: ev.TraderID,
			Type:     broker.RequestTypeApproval,
			Time:     ev.Time,
			Intent:   ev.Intent,
			Message:  ev.Message,
		}
		s.view.AddRequest(req)
	}
}

// AttachNewsEvents starts listening to news events in a goroutine.
func (s *BrokerService) AttachNewsEvents(events <-chan newsview.NewsEvent) {
	s.wg.Add(1)
	go s.runNewsEventListener(events)
}

func (s *BrokerService) runNewsEventListener(events <-chan newsview.NewsEvent) {
	defer s.wg.Done()

	for {
		select {
		case <-s.closed:
			return
		case _, ok := <-events:
			if !ok {
				return
			}
			// For now, we don't do anything with news events
			// This could be extended to show news in the broker view
		}
	}
}

// Requests returns the current broker requests.
func (s *BrokerService) Requests() []broker.Request {
	return s.view.Requests()
}

// PendingRequests returns unprocessed requests.
func (s *BrokerService) PendingRequests() []broker.Request {
	return s.view.PendingRequests()
}

// Close shuts down the broker service.
func (s *BrokerService) Close() {
	s.closeOnce.Do(func() {
		close(s.closed)
	})
	s.wg.Wait()
}
