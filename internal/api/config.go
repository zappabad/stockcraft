package api

type Config struct {
	CommandBuffer int // inbound command queue
	EventBuffer   int // internal engine->dispatcher event queue

	TradeTapeSize int // view tape capacity

	// If true, external Events() is best-effort (drops if consumer is slow).
	// If false, slow consumers can stall the dispatcher and eventually the engine.
	DropExternalEvents  bool
	ExternalEventBuffer int
}

func (c Config) withDefaults() Config {
	if c.CommandBuffer <= 0 {
		c.CommandBuffer = 4096
	}
	if c.EventBuffer <= 0 {
		c.EventBuffer = 8192
	}
	if c.TradeTapeSize <= 0 {
		c.TradeTapeSize = 10000
	}
	if c.ExternalEventBuffer <= 0 {
		c.ExternalEventBuffer = 4096
	}
	return c
}
