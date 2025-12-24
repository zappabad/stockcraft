package service

// Config holds configuration for the news service.
type Config struct {
	// TapeSize is the capacity of the news ring buffer.
	TapeSize int
	// EventBuffer is the size of the internal event channel.
	EventBuffer int
	// ExternalEventBuffer is the size of the external events channel.
	ExternalEventBuffer int
	// DropExternalEvents determines whether external event channel drops on overflow.
	DropExternalEvents bool
}

// DefaultConfig returns a Config with reasonable defaults.
func DefaultConfig() Config {
	return Config{
		TapeSize:            100,
		EventBuffer:         256,
		ExternalEventBuffer: 256,
		DropExternalEvents:  true,
	}
}
