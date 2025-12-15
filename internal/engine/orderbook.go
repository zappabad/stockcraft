package engine

import (
	"container/heap"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// -----------------------------------------------------------------------------
// Core types
// -----------------------------------------------------------------------------

type Side uint8

const (
	SideBuy Side = iota
	SideSell
)

func (s Side) String() string {
	switch s {
	case SideBuy:
		return "BUY"
	case SideSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

type OrderKind uint8

const (
	OrderKindLimit OrderKind = iota
	OrderKindMarket
)

func (ok OrderKind) String() string {
	switch ok {
	case OrderKindLimit:
		return "LIMIT"
	case OrderKindMarket:
		return "MARKET"
	default:
		return "UNKNOWN"
	}
}

type PriceTicks int64
type Size int64

func (p *PriceTicks) String() string {
	return strconv.FormatInt(int64(*p), 10)
}

func (s *Size) String() string {
	return strconv.FormatInt(int64(*s), 10)
}

type Order struct {
	ID     int64
	UserID int64
	Side   Side
	Kind   OrderKind
	Price  PriceTicks
	Size   Size
	Time   int64 // UnixNano

	// internal links for FIFO queue per level
	level *level
	prev  *Order
	next  *Order
}

func (o *Order) IsFilled() bool {
	return o.Size <= 0
}

type Match struct {
	Bid        *Order
	Ask        *Order
	Price      PriceTicks
	SizeFilled Size
}

type Trade struct {
	Price     PriceTicks
	Size      Size
	TakerSide Side
	Time      int64
}

// -----------------------------------------------------------------------------
// Price level + FIFO queue
// -----------------------------------------------------------------------------

type level struct {
	Price       PriceTicks
	head, tail  *Order
	TotalVolume Size
}

// append order at tail (does NOT change TotalVolume)
func (l *level) appendOrder(o *Order) {
	o.level = l
	o.prev = l.tail
	o.next = nil

	if l.tail != nil {
		l.tail.next = o
	} else {
		l.head = o
	}
	l.tail = o
}

// pop head order (does NOT change TotalVolume)
func (l *level) popHead() *Order {
	o := l.head
	if o == nil {
		return nil
	}
	next := o.next
	l.head = next
	if next != nil {
		next.prev = nil
	} else {
		l.tail = nil
	}
	o.prev = nil
	o.next = nil
	o.level = nil
	return o
}

// unlink specific order from list (does NOT change TotalVolume)
func (l *level) unlinkOrder(o *Order) {
	if o.prev != nil {
		o.prev.next = o.next
	} else {
		l.head = o.next
	}
	if o.next != nil {
		o.next.prev = o.prev
	} else {
		l.tail = o.prev
	}
	o.prev = nil
	o.next = nil
	o.level = nil
}

// -----------------------------------------------------------------------------
// Heap of price levels
// -----------------------------------------------------------------------------

type levelHeap struct {
	data  []*level
	index map[*level]int // level -> index in heap.data
	isBid bool           // true => max-heap by price; false => min-heap by price
}

func newLevelHeap(isBid bool) *levelHeap {
	h := &levelHeap{
		data:  []*level{},
		index: make(map[*level]int),
		isBid: isBid,
	}
	heap.Init(h)
	return h
}

func (h *levelHeap) Len() int { return len(h.data) }

func (h *levelHeap) Less(i, j int) bool {
	if h.isBid {
		// max-heap: highest price has priority
		return h.data[i].Price > h.data[j].Price
	}
	// min-heap: lowest price has priority
	return h.data[i].Price < h.data[j].Price
}

func (h *levelHeap) Swap(i, j int) {
	h.data[i], h.data[j] = h.data[j], h.data[i]
	h.index[h.data[i]] = i
	h.index[h.data[j]] = j
}

func (h *levelHeap) Push(x interface{}) {
	l := x.(*level)
	h.data = append(h.data, l)
	h.index[l] = len(h.data) - 1
}

func (h *levelHeap) Pop() interface{} {
	n := len(h.data)
	if n == 0 {
		return nil
	}
	l := h.data[n-1]
	h.data = h.data[:n-1]
	delete(h.index, l)
	return l
}

// remove arbitrary level from heap
func (h *levelHeap) removeLevel(l *level) {
	i, ok := h.index[l]
	if !ok {
		return
	}
	heap.Remove(h, i)
}

func (h *levelHeap) bestLevel() *level {
	if len(h.data) == 0 {
		return nil
	}
	return h.data[0]
}

// -----------------------------------------------------------------------------
// BookSide: one side of the book (bids or asks)
// -----------------------------------------------------------------------------

type bookSide struct {
	isBid  bool
	levels map[PriceTicks]*level // price → level
	lheap  *levelHeap
}

func newBookSide(isBid bool) *bookSide {
	return &bookSide{
		isBid:  isBid,
		levels: make(map[PriceTicks]*level),
		lheap:  newLevelHeap(isBid),
	}
}

func (bs *bookSide) bestLevel() *level {
	return bs.lheap.bestLevel()
}

func (bs *bookSide) getOrCreateLevel(price PriceTicks) *level {
	if l, ok := bs.levels[price]; ok {
		return l
	}
	l := &level{Price: price}
	bs.levels[price] = l
	heap.Push(bs.lheap, l)
	return l
}

func (bs *bookSide) removeLevel(l *level) {
	delete(bs.levels, l.Price)
	bs.lheap.removeLevel(l)
}

func (bs *bookSide) addRestingOrder(o *Order) {
	l := bs.getOrCreateLevel(o.Price)
	l.appendOrder(o)
	l.TotalVolume += o.Size
}

// cancel a resting order (assumes it is resting)
func (bs *bookSide) cancelRestingOrder(o *Order) {
	l := o.level
	if l == nil {
		return
	}
	// adjust volume before unlinking
	l.TotalVolume -= o.Size
	l.unlinkOrder(o)

	if l.TotalVolume <= 0 || l.head == nil {
		bs.removeLevel(l)
	}
}

// -----------------------------------------------------------------------------
// OrderBook: full book, both sides
// -----------------------------------------------------------------------------

type OrderBook struct {
	bids *bookSide
	asks *bookSide

	orders map[int64]*Order // ID → Order
	trades []*Trade

	mu sync.RWMutex
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		bids:   newBookSide(true),
		asks:   newBookSide(false),
		orders: make(map[int64]*Order),
		trades: []*Trade{},
	}
}

// -----------------------------------------------------------------------------
// Order factories (for trader code)
// -----------------------------------------------------------------------------

func now() int64 { return time.Now().UnixNano() }

func NewLimitOrder(id, userID int64, side Side, price PriceTicks, size Size) *Order {
	return &Order{
		ID:     id,
		UserID: userID,
		Side:   side,
		Kind:   OrderKindLimit,
		Price:  price,
		Size:   size,
		Time:   now(),
	}
}

func NewMarketOrder(id, userID int64, side Side, size Size) *Order {
	return &Order{
		ID:     id,
		UserID: userID,
		Side:   side,
		Kind:   OrderKindMarket,
		Price:  PriceTicks(0),
		Size:   size,
		Time:   now(),
	}
}

// -----------------------------------------------------------------------------
// Public API: submit / cancel / query
// -----------------------------------------------------------------------------

// SubmitLimitOrder: crossing behavior.
// - Try to match up to the limit price.
// - If any size remains, rest it at the limit.
func (ob *OrderBook) SubmitLimitOrder(o *Order) ([]Match, *Order, error) {
	if o.Kind != OrderKindLimit {
		return nil, nil, errors.New("SubmitLimitOrder: order must be limit")
	}

	ob.mu.Lock()
	defer ob.mu.Unlock()

	if existing, exists := ob.orders[o.ID]; exists {
		return nil, nil, errors.New(
			"SubmitLimitOrder: duplicate order ID=" +
				fmt.Sprint(o.ID) +
				" existing(user=" + fmt.Sprint(existing.UserID) +
				" side=" + existing.Side.String() +
				" price=" + fmt.Sprint(existing.Price) +
				" size=" + fmt.Sprint(existing.Size) + ")",
		)
	}

	// crossing part
	limit := o.Price
	matches := ob.matchOrderLocked(o, &limit)

	// rest any leftover size
	if !o.IsFilled() {
		side := ob.sideFor(o.Side)
		side.addRestingOrder(o)
		ob.orders[o.ID] = o
	}

	return matches, o, nil
}

// SubmitMarketOrder: match only, never rest.
func (ob *OrderBook) SubmitMarketOrder(o *Order) ([]Match, error) {
	if o.Kind != OrderKindMarket {
		return nil, errors.New("SubmitMarketOrder: order must be market")
	}

	ob.mu.Lock()
	defer ob.mu.Unlock()

	if _, exists := ob.orders[o.ID]; exists {
		return nil, errors.New("SubmitMarketOrder: duplicate order ID")
	}

	matches := ob.matchOrderLocked(o, nil)
	// market orders never go into orders map; they are either filled or left partially unfilled
	return matches, nil
}

// CancelOrder by ID. Returns true if something was canceled.
func (ob *OrderBook) CancelOrder(id int64) bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	o, ok := ob.orders[id]
	if !ok {
		return false
	}
	side := ob.sideFor(o.Side)
	side.cancelRestingOrder(o)
	delete(ob.orders, id)
	return true
}

// BestBid returns (price, size, ok)
func (ob *OrderBook) BestBid() (PriceTicks, Size, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	l := ob.bids.bestLevel()
	if l == nil {
		return 0, 0, false
	}
	return l.Price, l.TotalVolume, true
}

// BestAsk returns (price, size, ok)
func (ob *OrderBook) BestAsk() (PriceTicks, Size, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	l := ob.asks.bestLevel()
	if l == nil {
		return 0, 0, false
	}
	return l.Price, l.TotalVolume, true
}

// Snapshot of bids (best to worst)
type BookLevel struct {
	Price PriceTicks
	Size  Size
}

// BidsSnapshot returns a snapshot of all bid levels sorted best to worst.
func (ob *OrderBook) BidsSnapshot() []BookLevel {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	// copy heap data and sort manually (we don't mutate book heap)
	hdata := make([]*level, len(ob.bids.lheap.data))
	copy(hdata, ob.bids.lheap.data)

	// simple sort: highest price first
	// for a real system you might want a more efficient approach,
	// but this is only for inspection, not on the hot path.
	for i := 0; i < len(hdata); i++ {
		for j := i + 1; j < len(hdata); j++ {
			if hdata[j].Price > hdata[i].Price {
				hdata[i], hdata[j] = hdata[j], hdata[i]
			}
		}
	}

	out := make([]BookLevel, len(hdata))
	for i, l := range hdata {
		out[i] = BookLevel{Price: l.Price, Size: l.TotalVolume}
	}
	return out
}

// AsksSnapshot returns a snapshot of all ask levels sorted best to worst (lowest to highest).
func (ob *OrderBook) AsksSnapshot() []BookLevel {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	hdata := make([]*level, len(ob.asks.lheap.data))
	copy(hdata, ob.asks.lheap.data)

	// simple sort: lowest price first
	for i := 0; i < len(hdata); i++ {
		for j := i + 1; j < len(hdata); j++ {
			if hdata[j].Price < hdata[i].Price {
				hdata[i], hdata[j] = hdata[j], hdata[i]
			}
		}
	}

	out := make([]BookLevel, len(hdata))
	for i, l := range hdata {
		out[i] = BookLevel{Price: l.Price, Size: l.TotalVolume}
	}
	return out
}

// -----------------------------------------------------------------------------
// Internal helpers (protected by ob.mu)
// -----------------------------------------------------------------------------

func (ob *OrderBook) sideFor(s Side) *bookSide {
	if s == SideBuy {
		return ob.bids
	}
	return ob.asks
}

// matchOrderLocked matches an incoming order against the opposite side.
// limitPrice == nil => true market order (no price limit).
// limitPrice != nil => only match while best opposite price is acceptable.
// Assumes ob.mu is held.
func (ob *OrderBook) matchOrderLocked(o *Order, limitPrice *PriceTicks) []Match {
	var matches []Match

	opp := ob.asks
	if o.Side == SideSell {
		opp = ob.bids
	}

	for o.Size > 0 {
		best := opp.bestLevel()
		if best == nil {
			break
		}

		// limit price checks
		if limitPrice != nil {
			switch o.Side {
			case SideBuy:
				// buy limit: best ask must be <= limit
				if best.Price > *limitPrice {
					return matches
				}
			case SideSell:
				// sell limit: best bid must be >= limit
				if best.Price < *limitPrice {
					return matches
				}
			}
		}

		// consume orders at this best level
		for o.Size > 0 && best.head != nil {
			maker := best.head

			traded := o.Size
			if maker.Size < traded {
				traded = maker.Size
			}
			if traded <= 0 {
				// should not happen, but guard against infinite loops
				best.popHead()
				continue
			}

			o.Size -= traded
			maker.Size -= traded
			best.TotalVolume -= traded

			match := Match{
				Price:      best.Price,
				SizeFilled: traded,
			}
			if o.Side == SideBuy {
				match.Bid = o
				match.Ask = maker
			} else {
				match.Bid = maker
				match.Ask = o
			}
			matches = append(matches, match)

			// record trade
			ob.trades = append(ob.trades, &Trade{
				Price:     match.Price,
				Size:      match.SizeFilled,
				TakerSide: o.Side,
				Time:      now(),
			})

			if maker.IsFilled() {
				// maker fully consumed: remove from FIFO and orders map
				best.popHead()
				delete(ob.orders, maker.ID)
			} else {
				// taker must be fully filled if maker not, because traded = min
				// if maker still has size, o.Size must be zero
				if o.IsFilled() {
					break
				}
			}
		}

		if best.TotalVolume <= 0 || best.head == nil {
			opp.removeLevel(best)
		}

		if o.IsFilled() {
			break
		}
	}

	return matches
}

// -----------------------------------------------------------------------------
// Access to trades (optional)
// -----------------------------------------------------------------------------

func (ob *OrderBook) Trades() []*Trade {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	out := make([]*Trade, len(ob.trades))
	copy(out, ob.trades)
	return out
}
