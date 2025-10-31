package websocket_manager

import "context"

type ConnectionHandler interface {
	// OnConnect will be called at the beginning of the connection lifecycle once the MessageWriter is called.
	// It should finish quickly since it will block the connection.
	OnConnect(ctx context.Context)
	// OnDisconnect will be called at the end of the connection lifecycle.
	OnDisconnect(ctx context.Context)
	// OnMessage will be called in a separate goroutine, it is used to handle messages coming from the connection.
	// If an error is returned, the connection will be closed.
	OnMessage(ctx context.Context, payload []byte) error
	// MessageWriter should return a channel that will be used to send messages to the connection.
	// If an error is returned, the connection will be closed.
	MessageWriter(ctx context.Context) (<-chan []byte, error)
}
