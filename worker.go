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
			if conf.OnDisconnect != nil && wCtx.Err() == nil {
				conf.OnDisconnect.On(wCtx)
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
	if w.conf.OnConnect != nil && w.ctx.Err() == nil {
		go w.conf.OnConnect.On(w.ctx)
	}
	go w.handlePingPong()
	go w.readMessages()
	go w.writeMessages()
	return w.waitForClose()
}

func (w *Worker) handlePingPong() {
	_ = w.conn.SetReadDeadline(time.Now().Add(w.conf.PongTimeout))
	w.conn.SetPongHandler(func(_ string) error {
		w.conf.Logger.DebugContext(w.ctx, "pong received")
		return w.conn.SetReadDeadline(time.Now().Add(w.conf.PongTimeout))
	})

	ticker := time.NewTicker(w.conf.PingFrequency)
	defer ticker.Stop()

	w.conf.Logger.DebugContext(w.ctx, "pinger started", "conn", w.conn)
	for {
		select {
		case <-w.ctx.Done():
			w.conf.Logger.DebugContext(w.ctx, "pinger stopped")
			return

		case <-ticker.C:
			if err := w.conn.WriteControl(websocket.PingMessage, nil, w.conf.PingTimeout); err != nil {
				w.conf.Logger.ErrorContext(w.ctx, "failed to write ping message", "error", err)
				return
			}
			w.conf.Logger.DebugContext(w.ctx, "ping message sent")
		}
	}
}

func (w *Worker) writeMessages() {
	defer w.cancel()
	writeCh := make(chan []byte)
	go func() {
		defer w.cancel()
		if err := w.conf.MessageWriter.Write(w.ctx, writeCh); err != nil {
			w.conf.Logger.ErrorContext(w.ctx, "message writer failed", "error", err)
		}
	}()

	for {
		select {
		case <-w.ctx.Done():
			close(writeCh)
			return
		case payload, ok := <-writeCh:
			if !ok {
				return
			}
			if err := w.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				w.conf.Logger.ErrorContext(w.ctx, "failed to write payload", "error", err)
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
			if err := w.conf.MessageReader.Read(w.ctx, payload); err != nil {
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
