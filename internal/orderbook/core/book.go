package core

import "container/heap"

// internal resting order node (never exposed)
type restingOrder struct {
	id     OrderID
	userID UserID
	side   Side
	price  PriceTicks
	size   Size
	time   int64

	level *level
	prev  *restingOrder
	next  *restingOrder
}

func (o *restingOrder) isFilled() bool { return o.size <= 0 }

type level struct {
	price       PriceTicks
	head, tail  *restingOrder
	totalVolume Size
}

func (l *level) append(o *restingOrder) {
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

func (l *level) popHead() *restingOrder {
	o := l.head
	if o == nil {
		return nil
	}
	n := o.next
	l.head = n
	if n != nil {
		n.prev = nil
	} else {
		l.tail = nil
	}
	o.prev, o.next, o.level = nil, nil, nil
	return o
}

func (l *level) unlink(o *restingOrder) {
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
	o.prev, o.next, o.level = nil, nil, nil
}

// heap of levels
type levelHeap struct {
	data  []*level
	index map[*level]int
	isBid bool
}

func newLevelHeap(isBid bool) *levelHeap {
	h := &levelHeap{
		data:  []*level{},
		index: map[*level]int{},
		isBid: isBid,
	}
	heap.Init(h)
	return h
}

func (h *levelHeap) Len() int { return len(h.data) }
func (h *levelHeap) Less(i, j int) bool {
	if h.isBid {
		return h.data[i].price > h.data[j].price // max-heap for bids
	}
	return h.data[i].price < h.data[j].price // min-heap for asks
}
func (h *levelHeap) Swap(i, j int) {
	h.data[i], h.data[j] = h.data[j], h.data[i]
	h.index[h.data[i]] = i
	h.index[h.data[j]] = j
}
func (h *levelHeap) Push(x any) {
	l := x.(*level)
	h.data = append(h.data, l)
	h.index[l] = len(h.data) - 1
}
func (h *levelHeap) Pop() any {
	n := len(h.data)
	if n == 0 {
		return nil
	}
	l := h.data[n-1]
	h.data = h.data[:n-1]
	delete(h.index, l)
	return l
}
func (h *levelHeap) best() *level {
	if len(h.data) == 0 {
		return nil
	}
	return h.data[0]
}
func (h *levelHeap) removeLevel(l *level) {
	i, ok := h.index[l]
	if !ok {
		return
	}
	heap.Remove(h, i)
}

type bookSide struct {
	isBid  bool
	levels map[PriceTicks]*level
	h      *levelHeap
}

func newBookSide(isBid bool) *bookSide {
	return &bookSide{
		isBid:  isBid,
		levels: map[PriceTicks]*level{},
		h:      newLevelHeap(isBid),
	}
}

func (bs *bookSide) bestLevel() *level { return bs.h.best() }

func (bs *bookSide) getOrCreate(price PriceTicks) *level {
	if l, ok := bs.levels[price]; ok {
		return l
	}
	l := &level{price: price}
	bs.levels[price] = l
	heap.Push(bs.h, l)
	return l
}

func (bs *bookSide) removeLevel(l *level) {
	delete(bs.levels, l.price)
	bs.h.removeLevel(l)
}

type orderBook struct {
	bids *bookSide
	asks *bookSide

	orders map[OrderID]*restingOrder // resting only
}

func newOrderBook() *orderBook {
	return &orderBook{
		bids:   newBookSide(true),
		asks:   newBookSide(false),
		orders: map[OrderID]*restingOrder{},
	}
}

func (ob *orderBook) sideFor(s Side) *bookSide {
	if s == SideBuy {
		return ob.bids
	}
	return ob.asks
}

func (ob *orderBook) addResting(o Order) *restingOrder {
	node := &restingOrder{
		id:     o.ID,
		userID: o.UserID,
		side:   o.Side,
		price:  o.Price,
		size:   o.Size,
		time:   o.Time,
	}
	side := ob.sideFor(o.Side)
	l := side.getOrCreate(o.Price)
	l.append(node)
	l.totalVolume += node.size
	ob.orders[node.id] = node
	return node
}

func (ob *orderBook) cancel(id OrderID) (*restingOrder, bool) {
	node, ok := ob.orders[id]
	if !ok {
		return nil, false
	}
	side := ob.sideFor(node.side)
	l := node.level
	if l != nil {
		l.totalVolume -= node.size
		l.unlink(node)
		if l.totalVolume <= 0 || l.head == nil {
			side.removeLevel(l)
		}
	}
	delete(ob.orders, id)
	return node, true
}
