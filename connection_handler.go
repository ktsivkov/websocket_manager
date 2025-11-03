package websocket_manager

import "context"

type ConnectionHandler interface {
	// OnConnect will be called at the beginning of the connection lifecycle once the MessageWriter is called.
	// It should finish quickly since it will block the connection.
	OnConnect(ctx context.Context)
	// OnDisconnect will be called at the end of the connection lifecycle.
	OnDisconnect(ctx context.Context)
	// OnMessage will be called in a separate goroutine, it is used to handle messages coming from the connection.
	OnMessage(ctx context.Context, payload []byte)
	// MessageWriter should return a channel that will be used to send messages to the connection.
	// If the channel is closed, the connection will be closed.
	// If an error occurs, the connection will be closed.
	// If the channel returns a Message with type gorilla/websocket.CloseMessage, the connection will be closed after writing the message.
	MessageWriter(ctx context.Context) (<-chan Message, error)
}
