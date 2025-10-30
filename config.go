package websocket_manager

import (
	"net/http"
	"time"
)

type Config struct {
	// WriteControlTimeout How long to wait for a Connection.WriteControl to complete before timing out.
	WriteControlTimeout time.Duration
	// PingFrequency How often to send ping messages to clients.
	PingFrequency time.Duration
	// PingTimeout How long to wait for a ping message from a client before timing out.
	PingTimeout time.Duration
	// PongTimeout How long to wait for a pong message to be sent to a client before timing out.
	PongTimeout time.Duration
	// MessageWriter Sends messages to clients, if an error occurs, the connection is closed.
	MessageWriter MessageWriter
	// MessageReader Reads messages from clients, if an error occurs, the connection is closed..
	MessageReader MessageReader
	// Upgrader Upgrades the http connection to a websocket connection.
	Upgrader Upgrader
	Logger   Logger
	// Middlewares Execute before the ping-pong and message handlers.
	Middlewares []Middleware
	// ResponseHeader Response header to be set on the websocket upgrade response.
	ResponseHeader http.Header
	// OnConnect runs when a client connects.
	OnConnect ConnectionEventHandler
	// OnDisconnect runs when a client disconnects.
	OnDisconnect ConnectionEventHandler
}
