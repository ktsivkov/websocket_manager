package websocket_manager

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type worker struct {
	conn             *websocket.Conn
	socket           Socket
	conf             *Config
	closed           *atomic.Bool
	hasRan           *atomic.Bool
	closeMessageSent *atomic.Bool
	closeCh          chan error
}

func (w *worker) run() error {
	if w.hasRan.Load() {
		return ErrWorkerAlreadyRun
	}
	w.hasRan.Store(true)

	w.socket.OnConnect()
	go w.readMessages()
	go w.writeMessages()

	defer close(w.closeCh)
	return <-w.closeCh
}

func (w *worker) writeMessages() {
	var pingTickerCh <-chan time.Time
	if w.conf.isPingPongConfigured() {
		// Setup ping pong handlers.
		_ = w.conn.SetReadDeadline(time.Now().Add(w.conf.PongTimeout))
		w.conn.SetPongHandler(func(_ string) error {
			return w.conn.SetReadDeadline(time.Now().Add(w.conf.PongTimeout))
		})
		pingTicker := time.NewTicker(w.conf.PingFrequency)
		defer pingTicker.Stop()
		pingTickerCh = pingTicker.C
	}

	// Setup message writer.
	writerCh := w.socket.WriterChannel()
	for {
		select {
		case <-pingTickerCh:
			if err := w.conf.PingMessage.Write(w.conn, w.conf.WriteTimeout); err != nil {
				w.Close(fmt.Errorf("%w: %w", ErrPingMessage, err), nil)
				return
			}
		case payload, ok := <-writerCh:
			if !ok {
				w.Close(ErrWriterChannelClosed, nil)
				return
			}

			if err := payload.Write(w.conn, w.conf.WriteTimeout); err != nil {
				w.Close(fmt.Errorf("%w: %w", ErrFailedToWrite, err), nil)
				return
			}

			if payload.Type() == websocket.CloseMessage {
				w.closeMessageSent.Store(true)
				_ = w.conn.SetReadDeadline(time.Now().Add(w.conf.GracePeriod))
				return
			}
		}
	}
}

func (w *worker) readMessages() {
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

		if w.closeMessageSent.Load() {
			continue
		}

		go w.socket.OnMessage(payload)
	}
}

func (w *worker) Close(cause error, clientCloseMessage *ClientCloseMessage) {
	if w.closed.Load() {
		return
	}
	w.closed.Store(true)
	if w.closeMessageSent.Load() {
		cause = fmt.Errorf("%w: %w", ErrCloseMessageSent, cause)
	}

	w.socket.OnDisconnect(clientCloseMessage)

	if err := w.conn.Close(); err != nil {
		cause = fmt.Errorf("%w: %w", err, cause)
	}

	w.closeCh <- cause
}
