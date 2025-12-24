package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/zappabad/stockcraft/internal/market"
	marketview "github.com/zappabad/stockcraft/internal/market/view"
	"github.com/zappabad/stockcraft/internal/orderbook/core"
	orderbookservice "github.com/zappabad/stockcraft/internal/orderbook/service"
	orderbookview "github.com/zappabad/stockcraft/internal/orderbook/view"
)

var ErrUnknownTicker = errors.New("unknown ticker")

// MarketService manages multiple orderbooks and provides aggregated market data.
type MarketService struct {
	cfg     Config
	tickers map[market.TickerID]market.Ticker
	books   map[market.TickerID]*orderbookservice.Service
	mview   *marketview.MarketView

	externalEvents chan marketview.MarketEvent
	droppedEvents  atomic.Int64

	closed    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewMarketService creates a new MarketService with the given tickers.
func NewMarketService(tickers []market.Ticker, cfg Config) *MarketService {
	if cfg.MarketEventBuffer <= 0 {
		cfg.MarketEventBuffer = DefaultConfig().MarketEventBuffer
	}

	s := &MarketService{
		cfg:            cfg,
		tickers:        make(map[market.TickerID]market.Ticker, len(tickers)),
		books:          make(map[market.TickerID]*orderbookservice.Service, len(tickers)),
		mview:          marketview.NewMarketView(),
		externalEvents: make(chan marketview.MarketEvent, cfg.MarketEventBuffer),
		closed:         make(chan struct{}),
	}

	// Create orderbook service for each ticker
	for _, t := range tickers {
		tid := t.TickerID()
		s.tickers[tid] = t
		s.books[tid] = orderbookservice.NewService(cfg.Book)
	}

	// Start event forwarders for each book
	for tid, book := range s.books {
		s.wg.Add(1)
		go s.runBookEventForwarder(tid, book)
	}

	return s
}

func (s *MarketService) runBookEventForwarder(tid market.TickerID, book *orderbookservice.Service) {
	defer s.wg.Done()

	events := book.Events()
	for {
		select {
		case <-s.closed:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}

			// Update market view
			s.mview.Apply(tid, ev, book)

			// Emit to external channel
			me := marketview.MarketEvent{
				Ticker: tid,
				Event:  ev,
			}

			if s.cfg.DropMarketEvents {
				select {
				case s.externalEvents <- me:
				default:
					s.droppedEvents.Add(1)
				}
			} else {
				select {
				case s.externalEvents <- me:
				case <-s.closed:
					return
				}
			}
		}
	}
}

// SubmitLimit submits a limit order to the specified ticker's orderbook.
func (s *MarketService) SubmitLimit(ctx context.Context, tid market.TickerID, userID core.UserID, side core.Side, price core.PriceTicks, size core.Size) (core.SubmitReport, error) {
	book, ok := s.books[tid]
	if !ok {
		return core.SubmitReport{}, ErrUnknownTicker
	}
	return book.SubmitLimit(ctx, userID, side, price, size)
}

// SubmitMarket submits a market order to the specified ticker's orderbook.
func (s *MarketService) SubmitMarket(ctx context.Context, tid market.TickerID, userID core.UserID, side core.Side, size core.Size) (core.SubmitReport, error) {
	book, ok := s.books[tid]
	if !ok {
		return core.SubmitReport{}, ErrUnknownTicker
	}
	return book.SubmitMarket(ctx, userID, side, size)
}

// Cancel cancels an order in the specified ticker's orderbook.
func (s *MarketService) Cancel(ctx context.Context, tid market.TickerID, orderID core.OrderID) (core.CancelReport, error) {
	book, ok := s.books[tid]
	if !ok {
		return core.CancelReport{}, ErrUnknownTicker
	}
	return book.Cancel(ctx, orderID)
}

// GetLevels returns the price levels for a ticker and side.
func (s *MarketService) GetLevels(tid market.TickerID, side core.Side) ([]orderbookview.Level, error) {
	book, ok := s.books[tid]
	if !ok {
		return nil, ErrUnknownTicker
	}
	return book.GetLevels(side), nil
}

// GetOrders returns the resting orders for a ticker and side.
func (s *MarketService) GetOrders(tid market.TickerID, side core.Side) ([]orderbookview.RestingOrder, error) {
	book, ok := s.books[tid]
	if !ok {
		return nil, ErrUnknownTicker
	}
	return book.GetOrders(side), nil
}

// GetTradesLast returns the last n trades for a ticker.
func (s *MarketService) GetTradesLast(tid market.TickerID, n int) ([]core.TradeEvent, error) {
	book, ok := s.books[tid]
	if !ok {
		return nil, ErrUnknownTicker
	}
	return book.GetTradesLast(n), nil
}

// Snapshot returns the current market snapshot across all tickers.
func (s *MarketService) Snapshot() marketview.MarketSnapshot {
	return s.mview.SnapshotWithBooks(s.books)
}

// Events returns the consolidated market events channel.
func (s *MarketService) Events() <-chan marketview.MarketEvent {
	return s.externalEvents
}

// DroppedEvents returns the count of dropped market events.
func (s *MarketService) DroppedEvents() int64 {
	return s.droppedEvents.Load()
}

// GetTickers returns all registered tickers.
func (s *MarketService) GetTickers() []market.Ticker {
	tickers := make([]market.Ticker, 0, len(s.tickers))
	for _, t := range s.tickers {
		tickers = append(tickers, t)
	}
	return tickers
}

// Close shuts down the market service and all orderbook services.
func (s *MarketService) Close() {
	s.closeOnce.Do(func() {
		close(s.closed)
	})

	// Close all books
	for _, book := range s.books {
		book.Close()
	}

	// Wait for forwarders to finish
	s.wg.Wait()

	// Close external events channel
	close(s.externalEvents)
}
