package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zappabad/stockcraft/internal/orderbook/core"
	"github.com/zappabad/stockcraft/internal/orderbook/view"
)

// command types
type cmdType int

const (
	cmdSubmitLimit cmdType = iota
	cmdSubmitMarket
	cmdCancel
)

type command struct {
	typ    cmdType
	userID core.UserID
	side   core.Side
	price  core.PriceTicks
	size   core.Size
	id     core.OrderID // for cancel
	respCh chan<- response
}

type response struct {
	submitReport core.SubmitReport
	cancelReport core.CancelReport
	err          error
}

// Service owns the orderbook core and view, providing thread-safe access.
type Service struct {
	cfg  Config
	core *core.Core
	view *view.BookView

	idGen atomic.Int64

	cmdCh          chan command
	internalEvents chan core.Event
	externalEvents chan core.Event

	droppedExternal atomic.Int64

	closed    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewService creates a new orderbook Service.
func NewService(cfg Config) *Service {
	if cfg.CommandBuffer <= 0 {
		cfg.CommandBuffer = DefaultConfig().CommandBuffer
	}
	if cfg.EventBuffer <= 0 {
		cfg.EventBuffer = DefaultConfig().EventBuffer
	}
	if cfg.TradeTapeSize <= 0 {
		cfg.TradeTapeSize = DefaultConfig().TradeTapeSize
	}
	if cfg.ExternalEventBuffer <= 0 {
		cfg.ExternalEventBuffer = DefaultConfig().ExternalEventBuffer
	}

	s := &Service{
		cfg:            cfg,
		core:           core.NewCore(),
		view:           view.NewBookView(cfg.TradeTapeSize),
		cmdCh:          make(chan command, cfg.CommandBuffer),
		internalEvents: make(chan core.Event, cfg.EventBuffer),
		externalEvents: make(chan core.Event, cfg.ExternalEventBuffer),
		closed:         make(chan struct{}),
	}

	// Initialize ID generator from current time
	s.idGen.Store(time.Now().UnixNano())

	// Start command processor
	s.wg.Add(1)
	go s.runCommandProcessor()

	// Start event dispatcher
	s.wg.Add(1)
	go s.runEventDispatcher()

	return s
}

func (s *Service) nextID() core.OrderID {
	return core.OrderID(s.idGen.Add(1))
}

func (s *Service) runCommandProcessor() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closed:
			return
		case cmd := <-s.cmdCh:
			s.processCommand(cmd)
		}
	}
}

func (s *Service) processCommand(cmd command) {
	var resp response

	switch cmd.typ {
	case cmdSubmitLimit:
		o := core.Order{
			ID:     s.nextID(),
			UserID: cmd.userID,
			Side:   cmd.side,
			Kind:   core.OrderKindLimit,
			Price:  cmd.price,
			Size:   cmd.size,
			Time:   time.Now().UnixNano(),
		}
		report, events, err := s.core.SubmitLimit(o)
		resp = response{submitReport: report, err: err}
		for _, ev := range events {
			s.emitEvent(ev)
		}

	case cmdSubmitMarket:
		o := core.Order{
			ID:     s.nextID(),
			UserID: cmd.userID,
			Side:   cmd.side,
			Kind:   core.OrderKindMarket,
			Size:   cmd.size,
			Time:   time.Now().UnixNano(),
		}
		report, events, err := s.core.SubmitMarket(o)
		resp = response{submitReport: report, err: err}
		for _, ev := range events {
			s.emitEvent(ev)
		}

	case cmdCancel:
		report, events, err := s.core.Cancel(cmd.id, time.Now().UnixNano())
		resp = response{cancelReport: report, err: err}
		for _, ev := range events {
			s.emitEvent(ev)
		}
	}

	if cmd.respCh != nil {
		cmd.respCh <- resp
	}
}

func (s *Service) emitEvent(ev core.Event) {
	// Always send to internal channel (blocking is ok, buffer should be sufficient)
	select {
	case s.internalEvents <- ev:
	case <-s.closed:
		return
	}
}

func (s *Service) runEventDispatcher() {
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
					s.droppedExternal.Add(1)
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

// SubmitLimit submits a limit order.
func (s *Service) SubmitLimit(ctx context.Context, userID core.UserID, side core.Side, price core.PriceTicks, size core.Size) (core.SubmitReport, error) {
	respCh := make(chan response, 1)
	cmd := command{
		typ:    cmdSubmitLimit,
		userID: userID,
		side:   side,
		price:  price,
		size:   size,
		respCh: respCh,
	}

	select {
	case <-s.closed:
		return core.SubmitReport{}, context.Canceled
	case <-ctx.Done():
		return core.SubmitReport{}, ctx.Err()
	case s.cmdCh <- cmd:
	}

	select {
	case <-s.closed:
		return core.SubmitReport{}, context.Canceled
	case <-ctx.Done():
		return core.SubmitReport{}, ctx.Err()
	case resp := <-respCh:
		return resp.submitReport, resp.err
	}
}

// SubmitMarket submits a market order.
func (s *Service) SubmitMarket(ctx context.Context, userID core.UserID, side core.Side, size core.Size) (core.SubmitReport, error) {
	respCh := make(chan response, 1)
	cmd := command{
		typ:    cmdSubmitMarket,
		userID: userID,
		side:   side,
		size:   size,
		respCh: respCh,
	}

	select {
	case <-s.closed:
		return core.SubmitReport{}, context.Canceled
	case <-ctx.Done():
		return core.SubmitReport{}, ctx.Err()
	case s.cmdCh <- cmd:
	}

	select {
	case <-s.closed:
		return core.SubmitReport{}, context.Canceled
	case <-ctx.Done():
		return core.SubmitReport{}, ctx.Err()
	case resp := <-respCh:
		return resp.submitReport, resp.err
	}
}

// Cancel cancels a resting order.
func (s *Service) Cancel(ctx context.Context, id core.OrderID) (core.CancelReport, error) {
	respCh := make(chan response, 1)
	cmd := command{
		typ:    cmdCancel,
		id:     id,
		respCh: respCh,
	}

	select {
	case <-s.closed:
		return core.CancelReport{}, context.Canceled
	case <-ctx.Done():
		return core.CancelReport{}, ctx.Err()
	case s.cmdCh <- cmd:
	}

	select {
	case <-s.closed:
		return core.CancelReport{}, context.Canceled
	case <-ctx.Done():
		return core.CancelReport{}, ctx.Err()
	case resp := <-respCh:
		return resp.cancelReport, resp.err
	}
}

// GetLevels returns aggregate levels for a side (from view).
func (s *Service) GetLevels(side core.Side) []view.Level {
	return s.view.Levels(side)
}

// GetOrders returns resting orders for a side (from view).
func (s *Service) GetOrders(side core.Side) []view.RestingOrder {
	return s.view.Orders(side)
}

// GetTradesLast returns the last n trades (from view).
func (s *Service) GetTradesLast(n int) []core.TradeEvent {
	return s.view.TradesLast(n)
}

// Events returns the external events channel for subscribers.
func (s *Service) Events() <-chan core.Event {
	return s.externalEvents
}

// DroppedExternalEvents returns the count of dropped external events.
func (s *Service) DroppedExternalEvents() int64 {
	return s.droppedExternal.Load()
}

// Close shuts down the service and waits for goroutines to finish.
func (s *Service) Close() {
	s.closeOnce.Do(func() {
		close(s.closed)
	})
	s.wg.Wait()
}
