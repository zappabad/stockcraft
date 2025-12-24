package broker

import "github.com/zappabad/stockcraft/internal/trader"

// RequestType indicates the type of broker request.
type RequestType int

const (
	RequestTypeApproval RequestType = iota
	RequestTypeInfo
)

// Request represents a request from a trader that needs broker attention.
type Request struct {
	TraderID  trader.TraderID
	Type      RequestType
	Time      int64
	Intent    *trader.OrderIntent
	Message   string
	Processed bool
}
