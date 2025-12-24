package api

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zappabad/stockcraft/internal/view"

	"github.com/zappabad/stockcraft/internal/engine"
)

var ErrClosed = errors.New("service closed")

type Service struct {
	cfg Config

	core *engine.Core
	view *view.BookView

	cmdCh   chan any
	evBus   chan engine.Event
	pubEvCh chan engine.Event

	closed  chan struct{}
	closeMu sync.Mutex
	wg      sync.WaitGroup

	idSeq        atomic.Int64
	droppedEvCnt atomic.Int64
}

func NewService(cfg Config) *Service {
	cfg = cfg.withDefaults()

	s := &Service{
		cfg:     cfg,
		core:    engine.NewCore(),
		view:    view.NewBookView(cfg.TradeTapeSize),
		cmdCh:   make(chan any, cfg.CommandBuffer),
		evBus:   make(chan engine.Event, cfg.EventBuffer),
		pubEvCh: make(chan engine.Event, cfg.ExternalEventBuffer),
		closed:  make(chan struct{}),
	}

	// start ID sequence from current time to reduce accidental collisions on restarts
	s.idSeq.Store(time.Now().UnixNano())

	s.wg.Add(2)
	go s.engineLoop()
	go s.dispatcherLoop()
	return s
}

func (s *Service) Close() {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	select {
	case <-s.closed:
		return
	default:
		close(s.closed)
		close(s.cmdCh)
	}
	s.wg.Wait()
}

func (s *Service) Events() <-chan engine.Event  { return s.pubEvCh }
func (s *Service) DroppedExternalEvents() int64 { return s.droppedEvCnt.Load() }

func (s *Service) GetLevels(side engine.Side) []view.Level {
	return s.view.Levels(side)
}

func (s *Service) GetOrders(side engine.Side) []view.RestingOrder {
	return s.view.Orders(side)
}

func (s *Service) GetTradesLast(n int) []engine.TradeEvent {
	return s.view.TradesLast(n)
}

type submitCmd struct {
	order engine.Order
	reply chan submitResp
}
type submitResp struct {
	report engine.SubmitReport
	err    error
}
type cancelCmd struct {
	id    engine.OrderID
	reply chan cancelResp
}
type cancelResp struct {
	report engine.CancelReport
	err    error
}

func (s *Service) SubmitLimit(
	ctx context.Context,
	userID engine.UserID,
	side engine.Side,
	price engine.PriceTicks,
	size engine.Size,
) (engine.SubmitReport, error) {
	if err := s.ensureOpen(ctx); err != nil {
		return engine.SubmitReport{}, err
	}

	id := engine.OrderID(s.idSeq.Add(1))
	o := engine.Order{
		ID:     id,
		UserID: userID,
		Side:   side,
		Kind:   engine.OrderKindLimit,
		Price:  price,
		Size:   size,
		Time:   time.Now().UnixNano(),
	}

	reply := make(chan submitResp, 1)
	cmd := submitCmd{order: o, reply: reply}

	if err := s.sendCmd(ctx, cmd); err != nil {
		return engine.SubmitReport{}, err
	}

	select {
	case r := <-reply:
		return r.report, r.err
	case <-ctx.Done():
		return engine.SubmitReport{}, ctx.Err()
	case <-s.closed:
		return engine.SubmitReport{}, ErrClosed
	}
}

func (s *Service) SubmitMarket(
	ctx context.Context,
	userID engine.UserID,
	side engine.Side,
	size engine.Size,
) (engine.SubmitReport, error) {
	if err := s.ensureOpen(ctx); err != nil {
		return engine.SubmitReport{}, err
	}

	id := engine.OrderID(s.idSeq.Add(1))
	o := engine.Order{
		ID:     id,
		UserID: userID,
		Side:   side,
		Kind:   engine.OrderKindMarket,
		Price:  0,
		Size:   size,
		Time:   time.Now().UnixNano(),
	}

	reply := make(chan submitResp, 1)
	cmd := submitCmd{order: o, reply: reply}

	if err := s.sendCmd(ctx, cmd); err != nil {
		return engine.SubmitReport{}, err
	}

	select {
	case r := <-reply:
		return r.report, r.err
	case <-ctx.Done():
		return engine.SubmitReport{}, ctx.Err()
	case <-s.closed:
		return engine.SubmitReport{}, ErrClosed
	}
}

func (s *Service) Cancel(ctx context.Context, id engine.OrderID) (engine.CancelReport, error) {
	if err := s.ensureOpen(ctx); err != nil {
		return engine.CancelReport{}, err
	}

	reply := make(chan cancelResp, 1)
	cmd := cancelCmd{id: id, reply: reply}

	if err := s.sendCmd(ctx, cmd); err != nil {
		return engine.CancelReport{}, err
	}

	select {
	case r := <-reply:
		return r.report, r.err
	case <-ctx.Done():
		return engine.CancelReport{}, ctx.Err()
	case <-s.closed:
		return engine.CancelReport{}, ErrClosed
	}
}

func (s *Service) ensureOpen(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return ErrClosed
	default:
		return nil
	}
}

func (s *Service) sendCmd(ctx context.Context, cmd any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return ErrClosed
	case s.cmdCh <- cmd:
		return nil
	}
}

// owns core state; produces internal event bus
func (s *Service) engineLoop() {
	defer s.wg.Done()
	defer close(s.evBus)

	for cmd := range s.cmdCh {
		switch c := cmd.(type) {
		case submitCmd:
			var (
				rep engine.SubmitReport
				evs []engine.Event
				err error
			)
			if c.order.Kind == engine.OrderKindLimit {
				rep, evs, err = s.core.SubmitLimit(c.order)
			} else {
				rep, evs, err = s.core.SubmitMarket(c.order)
			}

			// publish events to internal bus (view + public). This must be complete for correctness.
			for _, ev := range evs {
				s.evBus <- ev
			}
			c.reply <- submitResp{report: rep, err: err}

		case cancelCmd:
			rep, evs, err := s.core.Cancel(c.id, time.Now().UnixNano())
			for _, ev := range evs {
				s.evBus <- ev
			}
			c.reply <- cancelResp{report: rep, err: err}
		}
	}
}

// applies events to view; then publishes externally (optionally best-effort)
func (s *Service) dispatcherLoop() {
	defer s.wg.Done()
	defer close(s.pubEvCh)

	for ev := range s.evBus {
		// view is the authoritative query model for GetOrders/GetTrades
		s.view.Apply(ev)

		if s.cfg.DropExternalEvents {
			select {
			case s.pubEvCh <- ev:
			default:
				s.droppedEvCnt.Add(1)
			}
		} else {
			// NOTE: a slow consumer can eventually stall the engine.
			s.pubEvCh <- ev
		}
	}
}
