package websocket_manager

import (
	"context"
	"errors"
	"time"

	"github.com/gorilla/websocket"
)

func newWorker(conn *Connection, conf Config) *Worker {
	wCtx, cancel := context.WithCancel(conn.Context())
	return &Worker{
		ctx: wCtx,
		cancel: func() {
			if wCtx.Err() == nil { // Ensure it is called only the first time.
				conf.ConnectionHandler.OnDisconnect(wCtx)
			}
			cancel()
		},
		conn: conn,
		conf: conf,
	}
}

type Worker struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   *Connection
	conf   Config
}

func (w *Worker) Run() error {
	w.conf.ConnectionHandler.OnConnect(w.ctx)
	go w.readMessages()
	go w.writeMessages()

	<-w.ctx.Done() // Wait for the context to be done.
	err := w.conn.Close()
	if err != nil {
		w.conf.Logger.ErrorContext(w.ctx, "failed to close connection", "error", err)
		return err
	}

	return nil
}

func (w *Worker) writeMessages() {
	defer w.cancel() // Close the connection on exit.

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
	writeCh, err := w.conf.ConnectionHandler.MessageWriter(w.ctx)
	if err != nil {
		w.conf.Logger.ErrorContext(w.ctx, "failed to get message writer", "error", err)
		return
	}

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-pingTicker.C:
			if err := pingMsg.Write(w.conn); err != nil {
				w.conf.Logger.ErrorContext(w.ctx, "failed to write ping message", "error", err)
				return
			}
		case payload, ok := <-writeCh:
			if !ok {
				return
			}
			if err := payload.Write(w.conn); err != nil {
				w.conf.Logger.ErrorContext(w.ctx, "failed to write message", "error", err)
				return
			}
			if payload.Type() == websocket.CloseMessage {
				return
			}
		}
	}
}

func (w *Worker) readMessages() {
	for {
		_, payload, err := w.conn.ReadMessage()
		if err != nil {
			if errors.Is(err, ErrConnectionClosed) {
				return
			}

			w.conf.Logger.ErrorContext(w.ctx, "failed to read message", "error", err)
			return
		}

		go w.conf.ConnectionHandler.OnMessage(w.ctx, payload)
	}
}
