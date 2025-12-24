package service

// Config holds configuration for the broker service.
type Config struct {
	// RequestCapacity is the maximum number of requests to keep.
	RequestCapacity int
}

// DefaultConfig returns a Config with reasonable defaults.
func DefaultConfig() Config {
	return Config{
		RequestCapacity: 100,
	}
}
