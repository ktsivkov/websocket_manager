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
	return w.waitForClose()
}

func (w *Worker) setupPingPongHandlers() {
	_ = w.conn.SetReadDeadline(time.Now().Add(w.conf.PongTimeout))
	w.conn.SetPongHandler(func(_ string) error {
		w.conf.Logger.DebugContext(w.ctx, "pong received")
		return w.conn.SetReadDeadline(time.Now().Add(w.conf.PongTimeout))
	})
}

func (w *Worker) writeMessages() {
	defer w.cancel()

	w.setupPingPongHandlers()
	pingTicker := time.NewTicker(w.conf.PingFrequency)
	defer pingTicker.Stop()

	writeCh, err := w.conf.ConnectionHandler.MessageWriter(w.ctx)
	if err != nil {
		w.conf.Logger.ErrorContext(w.ctx, "message writer failed", "error", err)
		return
	}

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-pingTicker.C:
			if err := w.conn.WriteControl(websocket.PingMessage, nil, w.conf.PingTimeout); err != nil {
				w.conf.Logger.ErrorContext(w.ctx, "failed to write ping message", "error", err)
				return
			}
			w.conf.Logger.DebugContext(w.ctx, "ping message sent")
		case payload := <-writeCh:
			if err := w.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				if errors.Is(err, ErrConnectionClosed) {
					w.conf.Logger.WarnContext(w.ctx, "message writer failed to write payload due to closed connection", "error", err, "payload", string(payload))
					return
				}
				w.conf.Logger.ErrorContext(w.ctx, "message writer failed to write payload", "error", err, "payload", string(payload))
				return
			}
			w.conf.Logger.DebugContext(w.ctx, "message sent", "payload", string(payload))
		}
	}
}

func (w *Worker) readMessages() {
	defer w.cancel()

	for {
		_, payload, err := w.conn.ReadMessage()
		if err != nil {
			if errors.Is(err, ErrConnectionClosed) {
				return
			}

			w.conf.Logger.ErrorContext(w.ctx, "failed to read payload", "error", err)
			if err := w.conn.WriteControlClose(websocket.ClosePolicyViolation, errMsgMalformed, w.conf.WriteControlTimeout); err != nil {
				w.conf.Logger.ErrorContext(w.ctx, "failed to write control message", "error", err)
				return
			}
			return
		}

		w.conf.Logger.DebugContext(w.ctx, "message received", "payload", string(payload))
		go func() {
			if err := w.conf.ConnectionHandler.OnMessage(w.ctx, payload); err != nil {
				w.conf.Logger.ErrorContext(w.ctx, "message reader failed", "error", err)
				w.cancel()
			}
		}()
	}
}

func (w *Worker) waitForClose() error {
	<-w.ctx.Done()

	err := w.conn.Close()
	if err != nil {
		w.conf.Logger.ErrorContext(w.ctx, "failed to close connection", "error", err)
		return err
	}

	return nil
}
