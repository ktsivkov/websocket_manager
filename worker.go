package websocket_manager

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

func newWorker(conn *websocket.Conn, client Socket, conf *Config) *Worker {
	return &Worker{
		conn:    conn,
		client:  client,
		conf:    conf,
		closed:  &atomic.Bool{},
		hasRan:  &atomic.Bool{},
		closeCh: make(chan error, 1),
	}
}

type Worker struct {
	conn    *websocket.Conn
	client  Socket
	conf    *Config
	closed  *atomic.Bool
	hasRan  *atomic.Bool
	closeCh chan error
}

// Run starts the worker.
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
// Returns any error that occurs during the run.
func (w *Worker) Run() error {
	if w.hasRan.Load() {
		return ErrWorkerAlreadyRun
	}
	w.hasRan.Store(true)

	w.client.OnConnect()
	go w.readMessages()
	go w.writeMessages()

	defer close(w.closeCh)
	return <-w.closeCh
}

func (w *Worker) writeMessages() {
	// Setup ping pong handlers.
	_ = w.conn.SetReadDeadline(time.Now().Add(w.conf.PongTimeout))
	w.conn.SetPongHandler(func(_ string) error {
		return w.conn.SetReadDeadline(time.Now().Add(w.conf.PongTimeout))
	})
	pingTicker := time.NewTicker(w.conf.PingFrequency)
	defer pingTicker.Stop()
	pingMsg := &controlMessage{
		typ:     websocket.PingMessage,
		data:    nil,
		timeout: w.conf.PingTimeout,
	}

	// Setup message writer.
	writerCh := w.client.WriterChannel()
	for {
		select {
		case <-pingTicker.C:
			if err := pingMsg.Write(w.conn); err != nil {
				w.Close(fmt.Errorf("%w: %w", ErrPingMessage, err), nil)
				return
			}
		case payload, ok := <-writerCh:
			if !ok {
				w.Close(ErrWriterChannelClosed, nil)
				return
			}

			if err := payload.Write(w.conn); err != nil {
				w.Close(fmt.Errorf("%w: %w", ErrFailedToWrite, err), nil)
				return
			}

			if payload.Type() == websocket.CloseMessage {
				w.Close(ErrCloseMessageSent, nil)
				return
			}
		}
	}
}

func (w *Worker) readMessages() {
	for {
		_, payload, err := w.conn.ReadMessage()
		if err != nil {
			if isConnectionClosedError(err) {
				w.Close(fmt.Errorf("%w: %w", ErrConnectionClosed, err), nil)
				return
			}
			if isTimeoutExceededError(err) { // At the moment, only pong timeout is supported; Therefore, it is safe to assume that it is a pong timeout.
				w.Close(fmt.Errorf("%w: %w", ErrPongTimeoutExceeded, err), nil)
				return
			}

			clientCloseMessage := clientCloseMessageFromError(err)
			if clientCloseMessage != nil {
				w.Close(ErrCloseMessageReceived, clientCloseMessage)
				return
			}

			w.Close(fmt.Errorf("%w: %w", ErrFailedToRead, err), clientCloseMessage)
			return
		}

		go w.client.OnMessage(payload)
	}
}

func (w *Worker) Close(cause error, clientCloseMessage *ClientCloseMessage) {
	if w.closed.Load() {
		return
	}
	w.closed.Store(true)

	w.client.OnDisconnect(clientCloseMessage)

	if err := w.conn.Close(); err != nil {
		cause = fmt.Errorf("%w: %w", err, cause)
	}

	w.closeCh <- cause
}
