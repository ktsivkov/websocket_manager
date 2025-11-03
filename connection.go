package websocket_manager

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionWithContext returns a new connection with the given context.
func ConnectionWithContext(ctx context.Context, conn *Connection) *Connection {
	return &Connection{
		ctx:  ctx,
		Conn: conn.Conn,
	}
}

type Connection struct {
	ctx context.Context
	*websocket.Conn
}

func (c *Connection) Context() context.Context {
	return c.ctx
}

// Close see https://pkg.go.dev/github.com/gorilla/websocket#Conn.Close
// Returns ErrAlreadyClosed if the connection is already closed.
func (c *Connection) Close() error {
	err := c.Conn.Close()
	if err == nil {
		return nil
	}

	if errors.Is(err, net.ErrClosed) {
		return fmt.Errorf("%w: %w", ErrAlreadyClosed, err)
	}

	return err
}

// ReadJSON see https://pkg.go.dev/github.com/gorilla/websocket#Conn.ReadJSON
// Returns ErrConnectionClosed if the connection is closed.
func (c *Connection) ReadJSON(v interface{}) error {
	if err := c.Conn.ReadJSON(v); err != nil {
		if isConnectionClosedError(err) {
			return fmt.Errorf("%w: %w", ErrConnectionClosed, err)
		}
		return err
	}
	return nil
}

// ReadMessage see https://pkg.go.dev/github.com/gorilla/websocket#Conn.ReadMessage
// Returns ErrConnectionClosed if the connection is closed.
func (c *Connection) ReadMessage() (int, []byte, error) {
	msgType, buf, err := c.Conn.ReadMessage()
	if err != nil {
		if isConnectionClosedError(err) {
			return 0, nil, fmt.Errorf("%w: %w", ErrConnectionClosed, err)
		}
		return 0, nil, err
	}

	return msgType, buf, nil
}

// WriteJSON see https://pkg.go.dev/github.com/gorilla/websocket#Conn.WriteJSON
// Returns ErrConnectionClosed if the connection is closed.
func (c *Connection) WriteJSON(v interface{}) error {
	if err := c.Conn.WriteJSON(v); err != nil {
		if isConnectionClosedError(err) {
			return fmt.Errorf("%w: %w", ErrConnectionClosed, err)
		}
		return err
	}

	return nil
}

// WriteMessage see https://pkg.go.dev/github.com/gorilla/websocket#Conn.WriteMessage
// Returns ErrConnectionClosed if the connection is closed.
func (c *Connection) WriteMessage(messageType int, data []byte) error {
	if err := c.Conn.WriteMessage(messageType, data); err != nil {
		if isConnectionClosedError(err) {
			return fmt.Errorf("%w: %w", ErrConnectionClosed, err)
		}
		return err
	}

	return nil
}

// WritePreparedMessage see https://pkg.go.dev/github.com/gorilla/websocket#Conn.WritePreparedMessage
// Returns ErrConnectionClosed if the connection is closed.
func (c *Connection) WritePreparedMessage(pm *websocket.PreparedMessage) error {
	if err := c.Conn.WritePreparedMessage(pm); err != nil {
		if isConnectionClosedError(err) {
			return fmt.Errorf("%w: %w", ErrConnectionClosed, err)
		}
		return err
	}

	return nil
}

// WriteControl see https://pkg.go.dev/github.com/gorilla/websocket#Conn.WriteControl
// timeout is the maximum time to wait for the writing to complete.
// Returns ErrConnectionClosed if the connection is closed.
// Returns ErrTimeoutExceeded if the writing timed out.
func (c *Connection) WriteControl(messageType int, data []byte, timeout time.Duration) error {
	if err := c.Conn.WriteControl(messageType, data, time.Now().Add(timeout)); err != nil {
		if isConnectionClosedError(err) {
			return fmt.Errorf("%w: %w", ErrConnectionClosed, err)
		}
		if isTimeoutExceededError(err) {
			return fmt.Errorf("%w: %w", ErrTimeoutExceeded, err)
		}
		return err
	}

	return nil
}

func isConnectionClosedError(err error) bool {
	return errors.As(err, new(*websocket.CloseError)) || errors.Is(err, net.ErrClosed)
}

func isTimeoutExceededError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
