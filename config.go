package websocket_manager

import (
	"time"
)

type Config struct {
	// PingFrequency How often to send ping messages to clients.
	PingFrequency time.Duration
	// PingTimeout How long to wait for a ping message to be written to a client before timing out.
	PingTimeout time.Duration
	// PongTimeout How long to wait for a pong message to be sent to a client before timing out.
	PongTimeout time.Duration
}
