package websocket_manager

import (
	"net/http"
	"time"
)

type Config struct {
	// PingFrequency How often to send ping messages to clients.
	PingFrequency time.Duration
	// PingTimeout How long to wait for a ping message from a client before timing out.
	PingTimeout time.Duration
	// PongTimeout How long to wait for a pong message to be sent to a client before timing out.
	PongTimeout time.Duration
	// Upgrader Upgrades the http connection to a websocket connection.
	Upgrader Upgrader
	Logger   Logger
	// Middlewares Execute before the ping-pong and message handlers.
	Middlewares []Middleware
	// ResponseHeader Response header to be set on the websocket upgrade response.
	ResponseHeader http.Header
	// ConnectionHandler Handles the connection lifecycle.
	ConnectionHandler ConnectionHandler
}
