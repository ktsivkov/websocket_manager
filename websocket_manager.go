package websocket_manager

import (
	"fmt"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// Run starts the websocket.
// Returns ErrWorkerAlreadyRun if the worker has already run.
// Returns ErrPingMessage if it fails to write a ping message.
// Returns ErrWriterChannelClosed if the WriterChannel of the Socket is closed.
// Returns ErrFailedToWrite if it fails to write a message.
// Returns ErrWriteTimeoutExceeded if the write timeout is exceeded.
// Returns ErrCloseMessageSent if the Socket sends a CloseMessage through the WriterChannel.
// Returns ErrCloseMessageReceived if the Socket receives a CloseMessage from the connection.
// Returns ErrFailedToRead if it fails to read a message.
// Returns ErrPongTimeoutExceeded if the pong timeout is exceeded.
// Returns ErrConnectionClosed if the connection is closed.
// Returns ErrConfigPartialPingConfiguration if the ping configuration is incomplete.
// Returns ErrConfigBadPingFrequency if the Config.PongTimeout is less or equal to Config.PingFrequency + Config.WriteTimeout.
// Returns any error that occurs during the run.
func Run(
	conn *websocket.Conn,
	socketCreator SocketCreator,
	conf *Config,
) error {
	if err := conf.validate(); err != nil {
		if connCloseErr := conn.Close(); connCloseErr != nil {
			return fmt.Errorf("%w: %w", err, connCloseErr)
		}
		return err
	}

	socket, err := socketCreator.Create()
	if err != nil {
		if connCloseErr := conn.Close(); connCloseErr != nil {
			return fmt.Errorf("%w: %w", err, connCloseErr)
		}
		return err
	}

	w := &worker{
		conn:                 conn,
		socket:               socket,
		conf:                 conf,
		closed:               &atomic.Bool{},
		hasRan:               &atomic.Bool{},
		serverInitiatedClose: &atomic.Bool{},
		closeCh:              make(chan error, 1),
	}

	return w.run()
}
