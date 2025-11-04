package websocket_manager

import (
	"context"
	"fmt"
	"net/http"
)

// New creates a new WebsocketManager.
// The spawned connections derive their context from the context passed to New.
// It is best to use context.Background() for the context. And implement a graceful shutdown by sending a CloseMessage to the open connections.
func New(ctx context.Context, conf Config) *WebsocketManager {
	return &WebsocketManager{
		ctx:  ctx,
		conf: conf,
	}
}

type WebsocketManager struct {
	ctx  context.Context
	conf Config
}

// Upgrade upgrades the connection to a websocket.
// Passes the context values to the ctx of the connection.
func (m *WebsocketManager) Upgrade(w http.ResponseWriter, r *http.Request, contextValues ...ContextValue) (*Worker, error) {
	conn, err := m.conf.Upgrader.Upgrade(w, r, m.conf.ResponseHeader)
	if err != nil {
		return nil, err
	}

	ctx := m.ctx
	for _, val := range contextValues {
		ctx = context.WithValue(ctx, val.Key, val.Val)
	}

	res := &Connection{
		ctx:  ctx,
		Conn: conn,
	}

	for _, middleware := range m.conf.Middlewares {
		newConn, err := middleware.Apply(res)
		if err != nil {
			middlewareErr := fmt.Errorf("%w: %w", ErrMiddlewareFailed, err)
			if closeErr := res.Close(); closeErr != nil {
				return nil, fmt.Errorf("%w: %w", ErrConnectionCloseError, fmt.Errorf("%w: %w", closeErr, middlewareErr))
			}
			return nil, middlewareErr
		}
		res = newConn
	}

	return newWorker(res, &m.conf), nil
}

// UpgradeAndRunAsync upgrades the connection to a websocket and runs it in a goroutine.
// See Upgrade and Worker.Run for more details.
func (m *WebsocketManager) UpgradeAndRunAsync(w http.ResponseWriter, r *http.Request, contextValues ...ContextValue) error {
	worker, err := m.Upgrade(w, r, contextValues...)
	if err != nil {
		return err
	}

	go func() {
		if err := worker.Run(); err != nil {
			m.conf.OnError("worker failed", err)
		}
	}()

	return nil
}
