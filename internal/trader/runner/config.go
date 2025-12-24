package runner

import "time"

// Config holds configuration for the trader runner.
type Config struct {
	// TickInterval is the interval between strategy ticks.
	TickInterval time.Duration
	// EventBuffer is the size of the trader events channel.
	EventBuffer int
	// DropEvents determines whether the events channel drops on overflow.
	DropEvents bool
}

// DefaultConfig returns a Config with reasonable defaults.
func DefaultConfig() Config {
	return Config{
		TickInterval: 100 * time.Millisecond,
		EventBuffer:  256,
		DropEvents:   true,
	}
}
