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
	validErr    error
	// PingFrequency How often to send ping messages to clients.
	// If 0, no ping messages will be sent.
	PingFrequency time.Duration
	// PongTimeout How long to wait for a pong message to be sent to a client before timing out.
	// If 0, no pong messages will be sent.
	PongTimeout time.Duration
	// WriteTimeout How long to wait for a message to be written to a client before timing out.
	WriteTimeout time.Duration
	// GracePeriod How long to wait for a client to acknowledge a close message before closing the connection.
	GracePeriod time.Duration
	mu          sync.Mutex
	validated   atomic.Bool
}

func (c *Config) isPingPongConfigured() bool {
	return c.PingMessage != nil && c.PingFrequency > 0 && c.PongTimeout > 0
}

func (c *Config) validate() error {
	if c.validated.CompareAndSwap(false, true) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.GracePeriod <= 0 {
			c.validErr = ErrConfigBadGracePeriod
			return c.validErr
		}
		if c.PingMessage != nil || c.PingFrequency != 0 || c.PongTimeout != 0 {
			if c.PingMessage == nil || c.PingFrequency == 0 || c.PongTimeout == 0 {
				c.validErr = ErrConfigPartialPingConfiguration
				return c.validErr
			}

			if c.PongTimeout <= c.PingFrequency+c.WriteTimeout {
				c.validErr = ErrConfigBadPingFrequency
				return c.validErr
			}
		}
	}

	return c.validErr
}
