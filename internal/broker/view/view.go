package view

import (
	"sync"

	"github.com/zappabad/stockcraft/internal/broker"
)

// BrokerView maintains the state of broker requests.
type BrokerView struct {
	mu       sync.RWMutex
	requests []broker.Request
	capacity int
}

// NewBrokerView creates a new BrokerView with the given capacity.
func NewBrokerView(capacity int) *BrokerView {
	if capacity <= 0 {
		capacity = 100
	}
	return &BrokerView{
		requests: make([]broker.Request, 0, capacity),
		capacity: capacity,
	}
}

// AddRequest adds a request to the view.
func (v *BrokerView) AddRequest(req broker.Request) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(v.requests) >= v.capacity {
		// Remove oldest
		v.requests = v.requests[1:]
	}
	v.requests = append(v.requests, req)
}

// Requests returns a copy of all pending requests.
func (v *BrokerView) Requests() []broker.Request {
	v.mu.RLock()
	defer v.mu.RUnlock()

	out := make([]broker.Request, len(v.requests))
	copy(out, v.requests)
	return out
}

// PendingRequests returns a copy of unprocessed requests.
func (v *BrokerView) PendingRequests() []broker.Request {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var out []broker.Request
	for _, req := range v.requests {
		if !req.Processed {
			out = append(out, req)
		}
	}
	return out
}
