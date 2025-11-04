package websocket_manager

type SocketCreator interface {
	// Create creates a new client instance.
	Create() (Socket, error)
}

type SocketCreatorFunc func() (Socket, error)

func (c SocketCreatorFunc) Create() (Socket, error) {
	return c()
}

type Socket interface {
	// OnConnect will be called at the beginning of the connection lifecycle once the MessageWriter is called.
	// It should finish quickly since it will block the connection.
	OnConnect()
	// OnDisconnect will be called at the end of the connection lifecycle.
	// The message will be nil if the connection was not upon a client request.
	OnDisconnect(msg *ClientCloseMessage)
	// OnMessage will be called in a separate goroutine, it is used to handle messages coming from the connection.
	OnMessage(payload []byte)
	// WriterChannel should return a channel that will be used to send messages to the connection.
	// If the channel is closed, the connection will be closed.
	// If an error occurs, the connection will be closed.
	// If the channel returns a Message with type gorilla/websocket.CloseMessage, the connection will be closed after writing the message.
	WriterChannel() <-chan Message
}
