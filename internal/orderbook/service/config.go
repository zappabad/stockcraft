package service

// Config holds configuration for the orderbook service.
type Config struct {
	// CommandBuffer is the size of the inbound command channel.
	CommandBuffer int
	// EventBuffer is the size of the internal authoritative event channel.
	EventBuffer int
	// TradeTapeSize is the capacity of the trade tape ring buffer.
	TradeTapeSize int
	// DropExternalEvents determines whether external event channel drops on overflow.
	DropExternalEvents bool
	// ExternalEventBuffer is the size of the external events channel.
	ExternalEventBuffer int
}

// DefaultConfig returns a Config with reasonable defaults.
func DefaultConfig() Config {
	return Config{
		CommandBuffer:       256,
		EventBuffer:         1024,
		TradeTapeSize:       1000,
		DropExternalEvents:  true,
		ExternalEventBuffer: 256,
	}
}
