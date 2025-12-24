package runner

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zappabad/stockcraft/internal/orderbook/core"
	"github.com/zappabad/stockcraft/internal/trader"
	"github.com/zappabad/stockcraft/internal/trader/strategy"
)

// Runner executes a trading strategy on a timer.
type Runner struct {
	cfg      Config
	traderID trader.TraderID
	strategy strategy.Strategy
	mr       strategy.MarketReader
	nr       strategy.NewsReader
	sender   strategy.OrderSender

	events        chan trader.TraderEvent
	droppedEvents atomic.Int64

	closed    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewRunner creates a new Runner.
func NewRunner(
	cfg Config,
	traderID trader.TraderID,
	strat strategy.Strategy,
	mr strategy.MarketReader,
	nr strategy.NewsReader,
	sender strategy.OrderSender,
) *Runner {
	if cfg.TickInterval <= 0 {
		cfg.TickInterval = DefaultConfig().TickInterval
	}
	if cfg.EventBuffer <= 0 {
		cfg.EventBuffer = DefaultConfig().EventBuffer
	}

	r := &Runner{
		cfg:      cfg,
		traderID: traderID,
		strategy: strat,
		mr:       mr,
		nr:       nr,
		sender:   sender,
		events:   make(chan trader.TraderEvent, cfg.EventBuffer),
		closed:   make(chan struct{}),
	}

	r.wg.Add(1)
	go r.run()

	return r
}

func (r *Runner) run() {
	defer r.wg.Done()
	defer close(r.events)

	ticker := time.NewTicker(r.cfg.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.closed:
			return
		case <-ticker.C:
			r.tick()
		}
	}
}

func (r *Runner) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.TickInterval)
	defer cancel()

	now := time.Now().UnixNano()

	intents, events := r.strategy.Step(ctx, now, r.mr, r.nr)

	// Execute order intents
	for _, intent := range intents {
		r.executeIntent(ctx, intent)
	}

	// Emit events
	for _, ev := range events {
		r.emitEvent(ev)
	}
}

func (r *Runner) executeIntent(ctx context.Context, intent trader.OrderIntent) {
	var err error

	switch intent.Kind {
	case core.OrderKindLimit:
		_, err = r.sender.SubmitLimit(ctx, intent.TickerID, core.UserID(r.traderID), intent.Side, intent.Price, intent.Size)
	case core.OrderKindMarket:
		_, err = r.sender.SubmitMarket(ctx, intent.TickerID, core.UserID(r.traderID), intent.Side, intent.Size)
	}

	if err != nil {
		r.emitEvent(trader.TraderEvent{
			TraderID: r.traderID,
			Time:     time.Now().UnixNano(),
			Type:     trader.TraderEventError,
			Message:  err.Error(),
		})
	}
}

func (r *Runner) emitEvent(ev trader.TraderEvent) {
	if r.cfg.DropEvents {
		select {
		case r.events <- ev:
		default:
			r.droppedEvents.Add(1)
		}
	} else {
		select {
		case r.events <- ev:
		case <-r.closed:
		}
	}
}

// Events returns the trader events channel.
func (r *Runner) Events() <-chan trader.TraderEvent {
	return r.events
}

// DroppedEvents returns the count of dropped events.
func (r *Runner) DroppedEvents() int64 {
	return r.droppedEvents.Load()
}

// Close shuts down the runner.
func (r *Runner) Close() {
	r.closeOnce.Do(func() {
		close(r.closed)
	})
	r.wg.Wait()
}
