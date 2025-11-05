package websocket_manager

import (
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	// PingMessage will be sent to clients based on the PingFrequency.
	// If nil, no ping messages will be sent.
	PingMessage Message
	// PingFrequency How often to send ping messages to clients.
	PingFrequency time.Duration
	// PongTimeout How long to wait for a pong message to be sent to a client before timing out.
	PongTimeout time.Duration
	// WriteTimeout How long to wait for a message to be written to a client before timing out.
	WriteTimeout time.Duration
	mu           sync.Mutex
	validated    atomic.Bool
	validErr     error
}

func (c *Config) isPingPongConfigured() bool {
	return c.PingMessage != nil && c.PingFrequency > 0 && c.PongTimeout > 0
}

func (c *Config) validate() error {
	if c.validated.CompareAndSwap(false, true) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.PingMessage != nil || c.PingFrequency != 0 || c.PongTimeout != 0 {
			if c.PingMessage == nil || c.PingFrequency == 0 || c.PongTimeout == 0 {
				c.validErr = ErrConfigPartialPingConfiguration
				return c.validErr
			}

			if c.PongTimeout >= c.PingFrequency+c.WriteTimeout {
				c.validErr = ErrConfigBadPingFrequency
				return c.validErr
			}
		}
	}

	return c.validErr
}
