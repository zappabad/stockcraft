package engine

import "errors"

var (
	ErrInvalidOrder = errors.New("invalid order")
	ErrDuplicateID  = errors.New("duplicate order id")
	ErrNotFound     = errors.New("order not found")
)

type Fill struct {
	MakerOrderID OrderID
	Price        PriceTicks
	Size         Size
}

type SubmitReport struct {
	OrderID   OrderID
	Remaining Size
	Fills     []Fill
	Rested    bool
}

type CancelReport struct {
	OrderID      OrderID
	CanceledSize Size
}

type Core struct {
	ob *OrderBook
}

func NewCore() *Core {
	return &Core{ob: NewOrderBook()}
}

func validateLimit(o Order) error {
	if o.Kind != OrderKindLimit {
		return ErrInvalidOrder
	}
	if o.ID == 0 || o.UserID == 0 {
		return ErrInvalidOrder
	}
	if o.Size <= 0 {
		return ErrInvalidOrder
	}
	if o.Price <= 0 {
		return ErrInvalidOrder
	}
	if o.Side != SideBuy && o.Side != SideSell {
		return ErrInvalidOrder
	}
	if o.Time <= 0 {
		return ErrInvalidOrder
	}
	return nil
}

func validateMarket(o Order) error {
	if o.Kind != OrderKindMarket {
		return ErrInvalidOrder
	}
	if o.ID == 0 || o.UserID == 0 {
		return ErrInvalidOrder
	}
	if o.Size <= 0 {
		return ErrInvalidOrder
	}
	if o.Side != SideBuy && o.Side != SideSell {
		return ErrInvalidOrder
	}
	if o.Time <= 0 {
		return ErrInvalidOrder
	}
	return nil
}

func (c *Core) SubmitLimit(o Order) (SubmitReport, []Event, error) {
	if err := validateLimit(o); err != nil {
		return SubmitReport{}, nil, err
	}
	if _, exists := c.ob.orders[o.ID]; exists {
		return SubmitReport{}, nil, ErrDuplicateID
	}

	remaining := o.Size
	limit := o.Price
	fills, evs := c.match(o, &limit)

	rested := false
	if remaining > 0 {
		o.Size = remaining
		c.ob.addResting(o)
		rested = true
		evs = append(evs, OrderRestedEvent{
			OrderID: o.ID, UserID: o.UserID, Side: o.Side,
			Price: o.Price, Size: o.Size, Time: o.Time,
		})
	}

	return SubmitReport{
		OrderID:   o.ID,
		Remaining: remaining,
		Fills:     fills,
		Rested:    rested,
	}, evs, nil
}

func (c *Core) SubmitMarket(o Order) (SubmitReport, []Event, error) {
	if err := validateMarket(o); err != nil {
		return SubmitReport{}, nil, err
	}
	if _, exists := c.ob.orders[o.ID]; exists {
		return SubmitReport{}, nil, ErrDuplicateID
	}

	remaining := o.Size
	fills, evs := c.match(o, nil)

	return SubmitReport{
		OrderID:   o.ID,
		Remaining: remaining,
		Fills:     fills,
		Rested:    false,
	}, evs, nil
}

func (c *Core) Cancel(id OrderID, now int64) (CancelReport, []Event, error) {
	if id == 0 || now <= 0 {
		return CancelReport{}, nil, ErrInvalidOrder
	}
	node, ok := c.ob.cancel(id)
	if !ok {
		return CancelReport{}, nil, ErrNotFound
	}
	ev := OrderRemovedEvent{
		OrderID:   node.id,
		Reason:    RemoveReasonCanceled,
		Remaining: node.size,
		Price:     node.price,
		Side:      node.side,
		UserID:    node.userID,
		Time:      now,
	}
	return CancelReport{OrderID: id, CanceledSize: node.size}, []Event{ev}, nil
}

// match consumes from opposite book. It mutates resting makers and emits events.
// remaining size is tracked via a closure variable so both SubmitLimit and SubmitMarket can use it.
func (c *Core) match(taker Order, limitPrice *PriceTicks) ([]Fill, []Event) {
	var (
		fills  []Fill
		events []Event
	)

	remaining := taker.Size

	opp := c.ob.asks
	if taker.Side == SideSell {
		opp = c.ob.bids
	}

	for remaining > 0 {
		best := opp.bestLevel()
		if best == nil {
			break
		}

		// limit checks
		if limitPrice != nil {
			switch taker.Side {
			case SideBuy:
				if best.price > *limitPrice {
					taker.Size = remaining
					return fills, events
				}
			case SideSell:
				if best.price < *limitPrice {
					taker.Size = remaining
					return fills, events
				}
			}
		}

		for remaining > 0 && best.head != nil {
			maker := best.head
			if maker.size <= 0 {
				// defensive: purge broken maker
				best.popHead()
				delete(c.ob.orders, maker.id)
				continue
			}

			traded := remaining
			if maker.size < traded {
				traded = maker.size
			}
			if traded <= 0 {
				// defensive: avoid infinite loop
				best.popHead()
				delete(c.ob.orders, maker.id)
				continue
			}

			remaining -= traded
			maker.size -= traded
			best.totalVolume -= traded

			fills = append(fills, Fill{
				MakerOrderID: maker.id,
				Price:        best.price,
				Size:         traded,
			})

			events = append(events, TradeEvent{
				Price:     best.price,
				Size:      traded,
				TakerSide: taker.Side,
				Time:      taker.Time,

				TakerOrderID: taker.ID,
				TakerUserID:  taker.UserID,
				MakerOrderID: maker.id,
				MakerUserID:  maker.userID,
			})

			if maker.isFilled() {
				best.popHead()
				delete(c.ob.orders, maker.id)

				events = append(events, OrderRemovedEvent{
					OrderID:   maker.id,
					Reason:    RemoveReasonFilled,
					Remaining: 0,
					Price:     maker.price,
					Side:      maker.side,
					UserID:    maker.userID,
					Time:      taker.Time,
				})
			} else {
				events = append(events, OrderReducedEvent{
					OrderID:   maker.id,
					Delta:     -traded,
					Remaining: maker.size,
					Price:     maker.price,
					Side:      maker.side,
					UserID:    maker.userID,
					MatchTime: taker.Time,
				})
			}
		}

		if best.totalVolume <= 0 || best.head == nil {
			opp.removeLevel(best)
		}
	}

	taker.Size = remaining
	return fills, events
}
