package websocket_manager

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

func New(ctx context.Context, conf Config) *WebsocketManager {
	return &WebsocketManager{
		ctx:  ctx,
		conf: conf,
		wg:   &sync.WaitGroup{},
	}
}

type WebsocketManager struct {
	ctx  context.Context
	conf Config
	wg   *sync.WaitGroup
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

	m.wg.Add(1)
	defer m.wg.Done()

	return newWorker(res, m.conf), nil
}

func (m *WebsocketManager) Wait() {
	m.wg.Wait()
}
